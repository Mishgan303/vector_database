package persistence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type FileSnapshotStore struct{ dir string }

var _ ports.SnapshotStore = (*FileSnapshotStore)(nil)

type snapshotRecord struct {
	LastLSN model.LSN       `json:"last_lsn"`
	Payload json.RawMessage `json:"payload"`
}

func NewFileSnapshotStore(dir string) *FileSnapshotStore { return &FileSnapshotStore{dir: dir} }

func (s *FileSnapshotStore) Save(ctx context.Context, snapshot model.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := encode(snapshotRecord{LastLSN: snapshot.LastLSN, Payload: snapshot.Payload})
	if err != nil {
		return err
	}
	_ = os.MkdirAll(s.dir, 0750)
	file, err := os.CreateTemp(s.dir, "snapshot-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, _ = file.Write(data)
	_ = file.Sync()
	_ = file.Close()
	if err = ctx.Err(); err != nil {
		return err
	}
	_ = os.Rename(file.Name(), filepath.Join(s.dir, "snapshot.json"))
	directory, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (s *FileSnapshotStore) LoadLatest(ctx context.Context) (model.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.Snapshot{}, err
	}
	data, err := os.ReadFile(filepath.Join(s.dir, "snapshot.json"))
	if err != nil {
		return model.Snapshot{}, err
	}
	var record snapshotRecord
	if err = decode(data, &record); err != nil {
		return model.Snapshot{}, err
	}
	return model.Snapshot{LastLSN: record.LastLSN, Payload: record.Payload}, nil
}
