package ports

import (
	"context"

	"vectordb/internal/model"
)

type WAL interface {
	Append(ctx context.Context, operation model.WALOperation) (model.LSN, error)
	Sync() error
	ReadAfter(ctx context.Context, lsn model.LSN, limit int) ([]model.WALEntry, error)
	TruncateThrough(ctx context.Context, lsn model.LSN) error
	Close() error
}
