package core

import (
	"context"

	"vectordb/internal/core/service"
)

// Config contains the settings used to open a database.
type Config = service.Config

// VectorDBService implements VectorDB.
type VectorDBService = service.VectorDBService

// CollectionRegistry holds the collections managed by the service.
type CollectionRegistry = service.CollectionRegistry

var _ VectorDB = (*service.VectorDBService)(nil)

// Open opens a database and restores its state from the snapshot and WAL.
func Open(ctx context.Context, config Config) (VectorDB, error) {
	db, err := service.Open(ctx, config)
	if err != nil {
		return nil, err
	}
	return db, nil
}
