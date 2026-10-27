package storage

import (
	"context"
	"sort"
	"sync"

	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type MemoryPointStore struct {
	mu     sync.RWMutex
	points map[string]map[model.PointRef]model.VectorPoint
}

var _ ports.PointStore = (*MemoryPointStore)(nil)

func NewMemoryPointStore() *MemoryPointStore {
	return &MemoryPointStore{points: make(map[string]map[model.PointRef]model.VectorPoint)}
}

func (s *MemoryPointStore) UpsertBatch(ctx context.Context, collection string, points []model.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.points[collection] == nil {
		s.points[collection] = make(map[model.PointRef]model.VectorPoint)
	}
	for _, point := range points {
		ref := model.PointRef{PartitionKey: point.PartitionKey, ID: point.ID}
		s.points[collection][ref] = model.ClonePoint(point)
	}
	return nil
}

func (s *MemoryPointStore) Get(ctx context.Context, collection string, ref model.PointRef) (model.VectorPoint, error) {
	if err := ctx.Err(); err != nil {
		return model.VectorPoint{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	point, ok := s.points[collection][ref]
	if !ok {
		return model.VectorPoint{}, model.DBError{Code: model.ErrorPointNotFound, Message: "point not found"}
	}
	return model.ClonePoint(point), nil
}

func (s *MemoryPointStore) DeleteBatch(ctx context.Context, collection string, refs []model.PointRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ref := range refs {
		delete(s.points[collection], ref)
	}
	if len(s.points[collection]) == 0 {
		delete(s.points, collection)
	}
	return nil
}

func (s *MemoryPointStore) Filter(ctx context.Context, collection string, filter *model.Filter) ([]model.PointRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conditions, err := normalizeFilter(filter)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	refs := make([]model.PointRef, 0)
	for ref, point := range s.points[collection] {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if matches(point.Metadata, conditions) {
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool { return model.LessRef(refs[i], refs[j]) })
	return refs, nil
}

func (s *MemoryPointStore) Descendants(ctx context.Context, collection string, root model.PointRef) ([]model.PointRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	children := make(map[model.PointRef][]model.PointRef)
	for ref, point := range s.points[collection] {
		if point.ParentID != nil {
			parent := model.PointRef{PartitionKey: point.PartitionKey, ID: *point.ParentID}
			children[parent] = append(children[parent], ref)
		}
	}
	seen := map[model.PointRef]struct{}{root: {}}
	queue := []model.PointRef{root}
	refs := make([]model.PointRef, 0)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := queue[0]
		queue = queue[1:]
		for _, ref := range children[current] {
			if _, ok := seen[ref]; !ok {
				seen[ref] = struct{}{}
				refs = append(refs, ref)
				queue = append(queue, ref)
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool { return model.LessRef(refs[i], refs[j]) })
	return refs, nil
}
