package ports_test

import (
	"context"
	"testing"

	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type pointStoreStub struct{}

func (pointStoreStub) UpsertBatch(context.Context, string, []model.VectorPoint) error { return nil }
func (pointStoreStub) Get(context.Context, string, model.PointRef) (model.VectorPoint, error) {
	return model.VectorPoint{}, nil
}
func (pointStoreStub) DeleteBatch(context.Context, string, []model.PointRef) error { return nil }
func (pointStoreStub) Filter(context.Context, string, *model.Filter) ([]model.PointRef, error) {
	return nil, nil
}
func (pointStoreStub) Descendants(context.Context, string, model.PointRef) ([]model.PointRef, error) {
	return nil, nil
}

type vectorIndexStub struct{}

func (vectorIndexStub) UpsertBatch(context.Context, []model.VectorPoint) error { return nil }
func (vectorIndexStub) DeleteBatch(context.Context, []model.PointRef) error    { return nil }
func (vectorIndexStub) Search(context.Context, model.IndexSearchRequest) ([]model.Neighbor, error) {
	return nil, nil
}

func TestStorageAndIndexContracts(t *testing.T) {
	var _ ports.PointStore = (*pointStoreStub)(nil)
	var _ ports.VectorIndex = (*vectorIndexStub)(nil)
}

type walStub struct{}

func (walStub) Append(context.Context, model.WALOperation) (model.LSN, error) { return 0, nil }
func (walStub) Sync() error                                                   { return nil }
func (walStub) ReadAfter(context.Context, model.LSN, int) ([]model.WALEntry, error) {
	return nil, nil
}
func (walStub) TruncateThrough(context.Context, model.LSN) error { return nil }
func (walStub) Close() error                                     { return nil }

type snapshotStoreStub struct{}

func (snapshotStoreStub) Save(context.Context, model.Snapshot) error { return nil }
func (snapshotStoreStub) LoadLatest(context.Context) (model.Snapshot, error) {
	return model.Snapshot{}, nil
}

type shardRouterStub struct{}

func (shardRouterStub) ShardFor(string) (model.ShardID, error) { return 0, nil }
func (shardRouterStub) AllShards() []model.ShardID             { return nil }

type replicatorStub struct{}

func (replicatorStub) Start(context.Context) error { return nil }
func (replicatorStub) Stop(context.Context) error  { return nil }

func TestDurabilityAndDistributionContracts(t *testing.T) {
	var _ ports.WAL = (*walStub)(nil)
	var _ ports.SnapshotStore = (*snapshotStoreStub)(nil)
	var _ ports.ShardRouter = (*shardRouterStub)(nil)
	var _ ports.Replicator = (*replicatorStub)(nil)
}
