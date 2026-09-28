package core

import (
	"context"

	"vectordb/internal/model"
)

type VectorDB interface {
	CreateCollection(ctx context.Context, config model.CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	GetCollection(ctx context.Context, name string) (model.CollectionConfig, error)
	ListCollections(ctx context.Context) ([]model.CollectionConfig, error)

	UpsertBatch(ctx context.Context, collection string, points []model.PointUpsert) (model.BatchResult, error)
	Get(ctx context.Context, collection string, ref model.PointRef) (model.VectorPoint, error)
	DeleteBatch(ctx context.Context, collection string, refs []model.PointRef) (model.BatchResult, error)
	Search(ctx context.Context, collection string, query model.SearchQuery) ([]model.SearchResult, error)

	Close(ctx context.Context) error
}
