package ports

import (
	"context"

	"vectordb/internal/model"
)

type SnapshotStore interface {
	Save(ctx context.Context, snapshot model.Snapshot) error
	LoadLatest(ctx context.Context) (model.Snapshot, error)
}
