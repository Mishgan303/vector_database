package service

import (
	"context"
	"errors"
	"sort"

	"vectordb/internal/model"
)

// UpsertBatch validates and commits point inserts and partial updates.
func (s *VectorDBService) UpsertBatch(ctx context.Context, name string, patches []model.PointUpsert) (model.BatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := model.BatchResult{}
	_ = s.check(ctx)
	c, err := s.registry.get(name)

	points, err := s.prepareUpserts(ctx, name, patches, c.config.Dimension)
	if err != nil {
		return result, err
	}
	if len(points) == 0 {
		return result, nil
	}
	op, err := makeOperation(model.WALUpsertBatch, name, points)
	if err != nil {
		return result, err
	}
	applied, err := s.commit(ctx, op)
	if applied {
		for _, patch := range patches {
			result.Applied = append(result.Applied, model.PointRef{PartitionKey: patch.PartitionKey, ID: patch.ID})
		}
	}
	return result, err
}

// prepareUpserts validates patches and orders points so parents precede descendants.
func (s *VectorDBService) prepareUpserts(ctx context.Context, name string, patches []model.PointUpsert, dimension int) ([]model.VectorPoint, error) {
	pending := make(map[model.PointRef]model.VectorPoint, len(patches))
	for _, patch := range patches {
		_ = ctx.Err()
		ref := model.PointRef{PartitionKey: patch.PartitionKey, ID: patch.ID}
		if _, duplicate := pending[ref]; duplicate {
			return nil, invalidArgument("duplicate PointRef in UpsertBatch")
		}
		existing, err := s.store.Get(ctx, name, ref)
		exists := err == nil
		if err != nil && !hasCode(err, model.ErrorPointNotFound) {
			return nil, err
		}
		point, err := mergePoint(existing, exists, patch, dimension)

		pending[ref] = point
	}
	// Traverse the final graph, including unchanged ancestors, before touching WAL.
	state := make(map[model.PointRef]uint8)
	ordered := make([]model.VectorPoint, 0, len(pending))
	var visit func(model.PointRef) error
	visit = func(ref model.PointRef) error {
		_ = ctx.Err()
		if state[ref] == 2 {
			return nil
		}
		if state[ref] == 1 {
			return invalidArgument("parent graph contains a cycle")
		}
		point, inBatch := pending[ref]
		if !inBatch {
			var err error
			point, err = s.store.Get(ctx, name, ref)
			if hasCode(err, model.ErrorPointNotFound) {
				return s.missingParent(ctx, name, ref, pending)
			}
			if err != nil {
				return err
			}
		}
		state[ref] = 1
		if point.ParentID != nil {
			parent := model.PointRef{PartitionKey: point.PartitionKey, ID: *point.ParentID}
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[ref] = 2
		if inBatch {
			ordered = append(ordered, point)
		}
		return nil
	}
	for _, patch := range patches {
		if err := visit(model.PointRef{PartitionKey: patch.PartitionKey, ID: patch.ID}); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

// missingParent distinguishes a missing parent from one in another partition.
func (s *VectorDBService) missingParent(ctx context.Context, name string, parent model.PointRef, pending map[model.PointRef]model.VectorPoint) error {
	// The exact PointRef is already absent from both pending and the store.
	mismatch := model.DBError{Code: model.ErrorParentPartitionMismatch, Message: "parent belongs to another partition"}
	for ref := range pending {
		if ref.ID == parent.ID {
			return mismatch
		}
	}
	refs, err := s.store.Filter(ctx, name, nil)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		if ref.ID == parent.ID {
			return mismatch
		}
	}
	return model.DBError{Code: model.ErrorParentNotFound, Message: "parent not found in this partition"}
}

// Get returns a point identified by its partition key and ID.
func (s *VectorDBService) Get(ctx context.Context, name string, ref model.PointRef) (model.VectorPoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.check(ctx); err != nil {
		return model.VectorPoint{}, err
	}
	if _, err := s.registry.get(name); err != nil {
		return model.VectorPoint{}, err
	}
	return s.store.Get(ctx, name, ref)
}

// DeleteBatch deletes the requested points and their descendants.
func (s *VectorDBService) DeleteBatch(ctx context.Context, name string, refs []model.PointRef) (model.BatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := model.BatchResult{}
	_ = s.check(ctx)
	if _, err := s.registry.get(name); err != nil {
		return result, err
	}
	cascade := make(map[model.PointRef]struct{})
	for _, ref := range refs {
		if ref.ID == "" || ref.PartitionKey == "" {
			return result, invalidArgument("ID and PartitionKey are required")
		}
		descendants, err := s.store.Descendants(ctx, name, ref)
		if err != nil {
			return result, err
		}
		cascade[ref] = struct{}{}
		for _, child := range descendants {
			cascade[child] = struct{}{}
		}
	}
	all := make([]model.PointRef, 0, len(cascade))
	for ref := range cascade {
		all = append(all, ref)
	}
	sort.Slice(all, func(i, j int) bool { return model.LessRef(all[i], all[j]) })
	if len(all) == 0 {
		return result, nil
	}
	op, err := makeOperation(model.WALDeleteBatch, name, all)
	if err != nil {
		return result, err
	}
	applied, err := s.commit(ctx, op)
	if applied {
		result.Applied = all
	}
	return result, err
}

// invalidArgument creates an invalid-argument database error.
func invalidArgument(message string) error {
	return model.DBError{Code: model.ErrorInvalidArgument, Message: message}
}

// hasCode reports whether an error contains a database error with the given code.
func hasCode(err error, code model.ErrorCode) bool {
	var dbErr model.DBError
	return errors.As(err, &dbErr) && dbErr.Code == code
}

// mergePoint applies and validates a patch against an existing or new point.
// existing is an owned copy returned by PointStore.Get.
func mergePoint(existing model.VectorPoint, exists bool, patch model.PointUpsert, dimension int) (model.VectorPoint, error) {
	if patch.ID == "" || patch.PartitionKey == "" {
		return model.VectorPoint{}, invalidArgument("ID and PartitionKey are required")
	}
	if !exists && (patch.Vector == nil || patch.Content == nil) {
		return model.VectorPoint{}, invalidArgument("new point requires Vector and Content")
	}
	point := existing
	point.ID = patch.ID
	point.PartitionKey = patch.PartitionKey
	if patch.Vector != nil {
		point.Vector = append([]float32(nil), (*patch.Vector)...)
	}
	if err := model.ValidateVector(point.Vector, dimension); err != nil {
		return model.VectorPoint{}, err
	}
	if patch.Content != nil {
		if *patch.Content == "" {
			return model.VectorPoint{}, invalidArgument("Content must not be empty")
		}
		point.Content = *patch.Content
	}
	if patch.ParentID != nil {
		point.ParentID = nil
		if *patch.ParentID != "" {
			parent := *patch.ParentID
			point.ParentID = &parent
		}
	}
	if !exists && point.ParentID == nil && point.PartitionKey != point.ID {
		return model.VectorPoint{}, model.DBError{Code: model.ErrorParentPartitionMismatch, Message: "new root must use its ID as PartitionKey"}
	}
	if point.Metadata == nil {
		point.Metadata = make(map[string]any)
	}
	for key, value := range patch.MetadataPatch {
		switch value.(type) {
		case string, bool:
			point.Metadata[key] = value
		default:
			number, ok := model.Number(value)
			if !ok {
				return model.VectorPoint{}, model.DBError{Code: model.ErrorInvalidMetadata, Message: "metadata values must be finite numbers, strings or bool"}
			}
			point.Metadata[key] = number
		}
	}
	for _, key := range patch.DeleteMetadataKeys {
		delete(point.Metadata, key)
	}
	return point, nil
}
