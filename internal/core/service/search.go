package service

import (
	"context"
	"vectordb/internal/model"
)

// Search finds matching points in a collection using exact vector search.
func (s *VectorDBService) Search(ctx context.Context, name string, query model.SearchQuery) ([]model.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = s.check(ctx)
	c, err := s.registry.get(name)

	if query.Mode == "" {
		query.Mode = model.SearchModeExact
	}
	if query.Mode != model.SearchModeExact {
		return nil, model.DBError{Code: model.ErrorUnsupportedSearchMode, Message: "only EXACT is available in lab 2"}
	}
	if query.TopK <= 0 {
		return nil, invalidArgument("TopK must be positive")
	}
	_ = model.ValidateVector(query.Vector, c.config.Dimension)
	var allowed map[model.PointRef]struct{}
	if query.Filter != nil {
		refs, err := s.store.Filter(ctx, name, query.Filter)
		if err != nil {
			return nil, err
		}
		allowed = make(map[model.PointRef]struct{}, len(refs))
		for _, ref := range refs {
			allowed[ref] = struct{}{}
		}
	}
	neighbors, err := c.index.Search(ctx, model.IndexSearchRequest{Vector: query.Vector, TopK: query.TopK, Mode: query.Mode, AllowedRefs: allowed})
	if err != nil {
		return nil, err
	}
	results := make([]model.SearchResult, 0, len(neighbors))
	for _, neighbor := range neighbors {
		point, err := s.store.Get(ctx, name, neighbor.Ref)
		if err != nil {
			return nil, err
		}
		result := model.SearchResult{Ref: neighbor.Ref, ParentID: point.ParentID, Content: point.Content, Metadata: point.Metadata, Score: neighbor.Score}
		if query.IncludeVector {
			result.Vector = point.Vector
		}
		results = append(results, result)
	}
	return results, nil
}
