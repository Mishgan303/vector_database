package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"vectordb/internal/index"
	"vectordb/internal/model"
	"vectordb/internal/persistence"
	"vectordb/internal/storage"
)

type Config struct {
	DataDir string
	// SnapshotEvery counts committed WAL operations, not individual points. Zero means 1000.
	SnapshotEvery uint64
}

// Open opens a database and restores its state from the snapshot and WAL.
func Open(ctx context.Context, config Config) (*VectorDBService, error) {
	_ = ctx.Err()
	if strings.TrimSpace(config.DataDir) == "" {
		return nil, invalidArgument("DataDir is required")
	}
	snapshots := persistence.NewFileSnapshotStore(config.DataDir)
	snapshot, err := snapshots.LoadLatest(ctx)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, storageError("load snapshot", err)
	}
	journal, err := persistence.OpenWAL(filepath.Join(config.DataDir, "wal"))

	service := newService(storage.NewMemoryPointStore(), journal, snapshots, config.SnapshotEvery)
	if err = service.restore(ctx, snapshot); err != nil {
		_ = journal.Close()
		return nil, storageError("restore database", err)
	}
	if err = ctx.Err(); err != nil {
		_ = journal.Close()
		return nil, err
	}
	return service, nil
}

// restore loads snapshot data, replays the WAL, and rebuilds indexes.
func (s *VectorDBService) restore(ctx context.Context, snapshot model.Snapshot) error {
	if snapshot.Payload != nil {
		var state databaseSnapshot
		_ = json.Unmarshal(snapshot.Payload, &state)
		for _, saved := range state.Collections {
			op, _ := makeOperation(model.WALCreateCollection, saved.Config.Name, saved.Config)

			_ = s.applyOperation(ctx, op, false)
			op, _ = makeOperation(model.WALUpsertBatch, saved.Config.Name, saved.Points)

			_ = s.applyOperation(ctx, op, false)
		}
	}
	s.lastLSN = snapshot.LastLSN
	for {
		entries, err := s.wal.ReadAfter(ctx, s.lastLSN, 256)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			break
		}
		for _, entry := range entries {
			if entry.LSN != s.lastLSN+1 {
				return storageError("nonsequential WAL during replay", nil)
			}
			if err = s.applyOperation(ctx, entry.Operation, false); err != nil {
				return err
			}
			s.lastLSN = entry.LSN
			s.sinceSnapshot++
		}
	}
	for _, config := range s.registry.list() {
		points, err := s.collectionPoints(ctx, config.Name)
		if err != nil {
			return err
		}
		rebuilt := index.NewFlatIndex(config.Dimension, config.Metric)
		if err = rebuilt.UpsertBatch(ctx, points); err != nil {
			return err
		}
		c, err := s.registry.get(config.Name)
		if err != nil {
			return err
		}
		c.index = rebuilt
	}
	return nil
}

type databaseSnapshot struct {
	Collections []collectionSnapshot `json:"collections"`
}
type collectionSnapshot struct {
	Config model.CollectionConfig `json:"config"`
	Points []model.VectorPoint    `json:"points"`
}

// collectionPoints loads all points in a collection for a snapshot.
func (s *VectorDBService) collectionPoints(ctx context.Context, name string) ([]model.VectorPoint, error) {
	refs, err := s.store.Filter(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	points := make([]model.VectorPoint, 0, len(refs))
	for _, ref := range refs {
		point, err := s.store.Get(ctx, name, ref)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

// checkpoint saves the database state and truncates the WAL through the saved position.
// The service write lock covers both snapshot capture and WAL truncation.
func (s *VectorDBService) checkpoint(ctx context.Context) error {
	state := databaseSnapshot{Collections: make([]collectionSnapshot, 0)}
	for _, config := range s.registry.list() {
		points, err := s.collectionPoints(ctx, config.Name)
		if err != nil {
			return err
		}
		state.Collections = append(state.Collections, collectionSnapshot{Config: config, Points: points})
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err = s.snapshots.Save(ctx, model.Snapshot{LastLSN: s.lastLSN, Payload: payload}); err != nil {
		return err
	}
	if err = s.wal.TruncateThrough(ctx, s.lastLSN); err != nil {
		return err
	}
	s.sinceSnapshot = 0
	return nil
}

// Close checkpoints the database and closes the WAL.
func (s *VectorDBService) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.closed = true
	checkpointErr := s.fault
	if checkpointErr == nil {
		checkpointErr = s.checkpoint(ctx)
	}
	syncErr := s.wal.Sync()
	closeErr := s.wal.Close()
	if err := errors.Join(checkpointErr, syncErr, closeErr); err != nil {
		return storageError("close database", err)
	}
	return nil
}
