package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"vectordb/internal/core"
	"vectordb/internal/model"
)

func main() {
	dir := flag.String("data", "./data", "каталог WAL и snapshot")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *dir, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, dir string, output io.Writer) (err error) {
	db, err := core.Open(ctx, core.Config{DataDir: dir})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close(context.Background())) }()
	_, err = db.GetCollection(ctx, "docs")
	existing := err == nil
	if err != nil {
		var dbErr model.DBError
		if !errors.As(err, &dbErr) || dbErr.Code != model.ErrorCollectionNotFound {
			return err
		}
		if err = db.CreateCollection(ctx, model.CollectionConfig{Name: "docs", Dimension: 2, Metric: model.MetricCosine, ShardCount: 1, ReplicationFactor: 1}); err != nil {
			return err
		}
	}
	parent := "guide"
	rootVector, childVector := []float32{1, 0}, []float32{0.8, 0.6}
	rootText, childText := "Векторные базы данных", "Поиск ближайших соседей"
	_, err = db.UpsertBatch(ctx, "docs", []model.PointUpsert{
		{ID: "guide", PartitionKey: "guide", Vector: &rootVector, Content: &rootText, MetadataPatch: map[string]any{"lang": "ru", "year": 2026}},
		{ID: "chunk-1", PartitionKey: "guide", ParentID: &parent, Vector: &childVector, Content: &childText, MetadataPatch: map[string]any{"lang": "ru", "year": 2026}},
	})
	if err != nil {
		return err
	}
	results, err := db.Search(ctx, "docs", model.SearchQuery{Vector: []float32{1, 0}, TopK: 2, Filter: &model.Filter{Conditions: []model.Condition{{Field: "lang", Operator: model.FilterEQ, Value: "ru"}}}})
	if err != nil {
		return err
	}
	if err = db.Close(ctx); err != nil {
		return err
	}
	restored, err := core.Open(ctx, core.Config{DataDir: dir})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, restored.Close(context.Background())) }()
	point, err := restored.Get(ctx, "docs", model.PointRef{PartitionKey: "guide", ID: "guide"})
	if err != nil {
		return err
	}
	report := struct {
		ExistingBeforeUpsert bool                 `json:"existing_before_upsert"`
		Results              []model.SearchResult `json:"results"`
		RestoredContent      string               `json:"restored_content"`
	}{existing, results, point.Content}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
