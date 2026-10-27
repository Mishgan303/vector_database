package index

import (
	"context"
	"sort"
	"sync"
	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type FlatIndex struct {
	mu        sync.RWMutex
	dimension int
	metric    model.DistanceMetric
	vectors   map[model.PointRef][]float32
}

var _ ports.VectorIndex = (*FlatIndex)(nil)

func NewFlatIndex(dimension int, metric model.DistanceMetric) *FlatIndex {
	return &FlatIndex{dimension: dimension, metric: metric, vectors: make(map[model.PointRef][]float32)}
}

func (f *FlatIndex) UpsertBatch(ctx context.Context, points []model.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range points {
		f.vectors[model.PointRef{PartitionKey: p.PartitionKey, ID: p.ID}] = append([]float32(nil), p.Vector...)
	}
	return nil
}

func (f *FlatIndex) DeleteBatch(ctx context.Context, refs []model.PointRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ref := range refs {
		delete(f.vectors, ref)
	}
	return nil
}

func (f *FlatIndex) Search(ctx context.Context, request model.IndexSearchRequest) ([]model.Neighbor, error) {
	_ = ctx.Err()
	if request.Mode != "" && request.Mode != model.SearchModeExact {
		return nil, model.DBError{Code: model.ErrorUnsupportedSearchMode, Message: "only EXACT search is supported"}
	}
	if request.TopK <= 0 {
		return nil, model.DBError{Code: model.ErrorInvalidArgument, Message: "TopK must be positive"}
	}
	if err := model.ValidateVector(request.Vector, f.dimension); err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	type candidate struct {
		ref   model.PointRef
		score float64
	}
	candidates := make([]candidate, 0)
	for ref, vector := range f.vectors {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if request.AllowedRefs != nil {
			if _, ok := request.AllowedRefs[ref]; !ok {
				continue
			}
		}
		candidates = append(candidates, candidate{ref: ref, score: distance(request.Vector, vector, f.metric)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return model.LessRef(candidates[i].ref, candidates[j].ref)
		}
		if f.metric == model.MetricL2 {
			return candidates[i].score < candidates[j].score
		}
		return candidates[i].score > candidates[j].score
	})
	candidates = candidates[:min(len(candidates), request.TopK)]
	neighbors := make([]model.Neighbor, 0, len(candidates))
	for _, candidate := range candidates {
		neighbors = append(neighbors, model.Neighbor{Ref: candidate.ref, Score: float32(candidate.score)})
	}
	return neighbors, nil
}
