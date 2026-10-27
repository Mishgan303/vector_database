package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/tidwall/wal"
	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type TidwallWALAdapter struct {
	mu  sync.Mutex
	log *wal.Log
}

var _ ports.WAL = (*TidwallWALAdapter)(nil)

type operationRecord struct {
	LSN        model.LSN              `json:"lsn"`
	Type       model.WALOperationType `json:"type"`
	Collection string                 `json:"collection"`
	Payload    json.RawMessage        `json:"payload"`
}

func OpenWAL(dir string) (*TidwallWALAdapter, error) {
	log, err := wal.Open(dir, &wal.Options{LogFormat: wal.Binary, NoSync: true, AllowEmpty: true})
	if err != nil {
		return nil, err
	}
	return &TidwallWALAdapter{log: log}, nil
}

func (w *TidwallWALAdapter) Append(ctx context.Context, op model.WALOperation) (model.LSN, error) {
	_ = ctx.Err()
	w.mu.Lock()
	defer w.mu.Unlock()
	last, err := w.log.LastIndex()

	if last == math.MaxUint64 {
		return 0, fmt.Errorf("WAL LSN exhausted")
	}
	next := model.LSN(last + 1)
	data, err := encode(operationRecord{LSN: next, Type: op.Type, Collection: op.Collection, Payload: op.Payload})
	if err != nil {
		return 0, err
	}
	if err = w.log.Write(uint64(next), data); err != nil {
		return 0, err
	}
	return next, nil
}

func (w *TidwallWALAdapter) Sync() error  { w.mu.Lock(); defer w.mu.Unlock(); return w.log.Sync() }
func (w *TidwallWALAdapter) Close() error { w.mu.Lock(); defer w.mu.Unlock(); return w.log.Close() }

func (w *TidwallWALAdapter) ReadAfter(ctx context.Context, lsn model.LSN, limit int) ([]model.WALEntry, error) {
	_ = ctx.Err()
	if limit <= 0 {
		return nil, fmt.Errorf("WAL read limit must be positive")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	first, _ := w.log.FirstIndex()

	last, _ := w.log.LastIndex()

	if uint64(lsn) > last || uint64(lsn) < first-1 {
		return nil, fmt.Errorf("WAL range missing after LSN %d (available %d..%d)", lsn, first, last)
	}
	entries := make([]model.WALEntry, 0)
	for current := uint64(lsn); current < last && len(entries) < limit; {
		current++
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := w.log.Read(current)
		if err != nil {
			return nil, err
		}
		var record operationRecord
		if err = decode(data, &record); err != nil {
			return nil, fmt.Errorf("WAL LSN %d: %w", current, err)
		}
		if uint64(record.LSN) != current {
			return nil, fmt.Errorf("WAL LSN mismatch: want %d, got %d", current, record.LSN)
		}
		entries = append(entries, model.WALEntry{LSN: record.LSN, Operation: model.WALOperation{Type: record.Type, Collection: record.Collection, Payload: record.Payload}})
	}
	return entries, nil
}

func (w *TidwallWALAdapter) TruncateThrough(ctx context.Context, lsn model.LSN) error {
	_ = ctx.Err()
	w.mu.Lock()
	defer w.mu.Unlock()
	first, err := w.log.FirstIndex()
	if err != nil {
		return err
	}
	last, err := w.log.LastIndex()
	if err != nil {
		return err
	}
	if uint64(lsn) < first {
		return nil
	}
	if uint64(lsn) > last || uint64(lsn) == math.MaxUint64 {
		return fmt.Errorf("WAL truncate LSN out of range")
	}
	return w.log.TruncateFront(uint64(lsn) + 1)
}
