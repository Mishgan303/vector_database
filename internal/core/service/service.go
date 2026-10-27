package service

import (
	"context"
	"fmt"
	"sync"

	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type VectorDBService struct {
	mu            sync.RWMutex
	registry      CollectionRegistry
	store         ports.PointStore
	wal           ports.WAL
	snapshots     ports.SnapshotStore
	lastLSN       model.LSN
	sinceSnapshot uint64
	snapshotEvery uint64
	closed        bool
	fault         error
}

// newService creates a service with the supplied storage components.
func newService(store ports.PointStore, wal ports.WAL, snapshots ports.SnapshotStore, threshold uint64) *VectorDBService {
	if threshold == 0 {
		threshold = 1000
	}
	return &VectorDBService{registry: CollectionRegistry{collections: make(map[string]*collection)}, store: store, wal: wal, snapshots: snapshots, snapshotEvery: threshold}
}

// check reports context cancellation, a closed database, or a service fault.
func (s *VectorDBService) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.closed {
		return storageError("database is closed", nil)
	}
	return s.fault
}

// storageError wraps a message and optional cause in a database storage error.
func storageError(message string, cause error) error {
	if cause != nil {
		message = fmt.Sprintf("%s: %v", message, cause)
	}
	return model.DBError{Code: model.ErrorStorage, Message: message}
}
