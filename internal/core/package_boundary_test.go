package core_test

import (
	"context"
	"testing"

	"vectordb/internal/core"
	"vectordb/internal/core/service"
)

func TestOpenUsesServiceImplementation(t *testing.T) {
	var _ core.VectorDB = (*service.VectorDBService)(nil)
	var _ *core.VectorDBService = (*service.VectorDBService)(nil)
	var _ *core.CollectionRegistry = (*service.CollectionRegistry)(nil)

	db, err := core.Open(context.Background(), core.Config{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := db.(*service.VectorDBService); !ok {
		t.Errorf("Open returned %T, want *service.VectorDBService", db)
	}
	if err := db.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOpenReturnsNilOnError(t *testing.T) {
	db, err := core.Open(context.Background(), core.Config{})
	if err == nil || db != nil {
		t.Fatalf("Open returned (%v, %v), want nil and an error", db, err)
	}
}
