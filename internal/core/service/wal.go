package service

import (
	"context"
	"encoding/json"
	"fmt"

	"vectordb/internal/index"
	"vectordb/internal/model"
)

// makeOperation encodes a WAL operation payload as JSON.
func makeOperation(kind model.WALOperationType, collection string, payload any) (model.WALOperation, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return model.WALOperation{}, storageError("encode operation", err)
	}
	return model.WALOperation{Type: kind, Collection: collection, Payload: data}, nil
}

// commit writes and syncs an operation to the WAL, then applies it.
// Once Sync succeeds, cancellation must not leave a durable operation unapplied.
func (s *VectorDBService) commit(ctx context.Context, op model.WALOperation) (bool, error) {
	_ = ctx.Err()
	lsn, err := s.wal.Append(ctx, op)

	if err = s.wal.Sync(); err != nil {
		s.fault = storageError("sync WAL; write outcome requires recovery", err)
		return false, s.fault
	}
	if err = s.applyOperation(context.Background(), op, true); err != nil {
		s.fault = storageError("apply durable operation; reopen required", err)
		return false, s.fault
	}
	s.lastLSN = lsn
	s.sinceSnapshot++
	if s.sinceSnapshot >= s.snapshotEvery {
		if err = s.checkpoint(context.Background()); err != nil {
			s.fault = storageError("write committed, but checkpoint failed; reopen required", err)
			return true, s.fault
		}
	}
	return true, nil
}

// applyOperation applies a committed or replayed operation without appending to the WAL.
func (s *VectorDBService) applyOperation(ctx context.Context, op model.WALOperation, updateIndex bool) error {
	switch op.Type {
	case model.WALCreateCollection:
		var config model.CollectionConfig
		_ = json.Unmarshal(op.Payload, &config)
		_ = validateConfig(config)
		if config.Name != op.Collection {
			return fmt.Errorf("collection name mismatch")
		}
		s.registry.mu.Lock()
		defer s.registry.mu.Unlock()
		if _, exists := s.registry.collections[config.Name]; exists {
			return fmt.Errorf("duplicate collection in WAL")
		}
		s.registry.collections[config.Name] = &collection{config: config, index: index.NewFlatIndex(config.Dimension, config.Metric)}
		return nil
	case model.WALDeleteCollection:
		if _, err := s.registry.get(op.Collection); err != nil {
			return err
		}
		refs, _ := s.store.Filter(ctx, op.Collection, nil)

		_ = s.store.DeleteBatch(ctx, op.Collection, refs)
		s.registry.mu.Lock()
		delete(s.registry.collections, op.Collection)
		s.registry.mu.Unlock()
		return nil
	case model.WALUpsertBatch:
		c, err := s.registry.get(op.Collection)

		var points []model.VectorPoint
		if err = json.Unmarshal(op.Payload, &points); err != nil {
			return err
		}
		for _, point := range points {
			if point.ID == "" || point.PartitionKey == "" || point.Content == "" {
				return fmt.Errorf("invalid stored point")
			}
			if err = model.ValidateVector(point.Vector, c.config.Dimension); err != nil {
				return err
			}
			for _, value := range point.Metadata {
				switch value.(type) {
				case string, bool:
				default:
					if _, ok := model.Number(value); !ok {
						return fmt.Errorf("invalid stored metadata")
					}
				}
			}
		}
		if err = s.store.UpsertBatch(ctx, op.Collection, points); err != nil {
			return err
		}
		if !updateIndex {
			return nil
		}
		return c.index.UpsertBatch(ctx, points)
	case model.WALDeleteBatch:
		c, err := s.registry.get(op.Collection)
		if err != nil {
			return err
		}
		var refs []model.PointRef
		if err = json.Unmarshal(op.Payload, &refs); err != nil {
			return err
		}
		if err = s.store.DeleteBatch(ctx, op.Collection, refs); err != nil {
			return err
		}
		if !updateIndex {
			return nil
		}
		return c.index.DeleteBatch(ctx, refs)
	default:
		return fmt.Errorf("unsupported WAL operation %q", op.Type)
	}
}
