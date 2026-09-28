package ports

import (
	"context"

	"vectordb/internal/model"
)

type VectorIndex interface {
	UpsertBatch(ctx context.Context, points []model.VectorPoint) error
	DeleteBatch(ctx context.Context, refs []model.PointRef) error
	Search(ctx context.Context, request model.IndexSearchRequest) ([]model.Neighbor, error)
}
