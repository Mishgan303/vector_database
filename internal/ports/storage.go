package ports

import (
	"context"

	"vectordb/internal/model"
)

type PointStore interface {
	UpsertBatch(ctx context.Context, collection string, points []model.VectorPoint) error
	Get(ctx context.Context, collection string, ref model.PointRef) (model.VectorPoint, error)
	DeleteBatch(ctx context.Context, collection string, refs []model.PointRef) error
	Filter(ctx context.Context, collection string, filter *model.Filter) ([]model.PointRef, error)
	Descendants(ctx context.Context, collection string, root model.PointRef) ([]model.PointRef, error)
}
