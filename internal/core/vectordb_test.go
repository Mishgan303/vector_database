package core_test

import (
	"context"
	"testing"

	"vectordb/internal/core"
	"vectordb/internal/model"
)

type stubVectorDB struct{}

func (stubVectorDB) CreateCollection(context.Context, model.CollectionConfig) error { return nil }
func (stubVectorDB) DeleteCollection(context.Context, string) error                 { return nil }
func (stubVectorDB) GetCollection(context.Context, string) (model.CollectionConfig, error) {
	return model.CollectionConfig{}, nil
}
func (stubVectorDB) ListCollections(context.Context) ([]model.CollectionConfig, error) {
	return nil, nil
}
func (stubVectorDB) UpsertBatch(context.Context, string, []model.PointUpsert) (model.BatchResult, error) {
	return model.BatchResult{}, nil
}
func (stubVectorDB) Get(context.Context, string, model.PointRef) (model.VectorPoint, error) {
	return model.VectorPoint{}, nil
}
func (stubVectorDB) DeleteBatch(context.Context, string, []model.PointRef) (model.BatchResult, error) {
	return model.BatchResult{}, nil
}
func (stubVectorDB) Search(context.Context, string, model.SearchQuery) ([]model.SearchResult, error) {
	return nil, nil
}
func (stubVectorDB) Close(context.Context) error { return nil }

func TestVectorDBContract(t *testing.T) {
	var _ core.VectorDB = (*stubVectorDB)(nil)
}
