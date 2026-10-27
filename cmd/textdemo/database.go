package main

import (
	"context"
	"errors"
	"fmt"

	"vectordb/internal/core"
	"vectordb/internal/model"
)

type document struct {
	ID      string
	Content string
}

// initializeDB opens the local database and restores saved data.
func initializeDB(ctx context.Context, dataDir string) (core.VectorDB, error) {
	return core.Open(ctx, core.Config{DataDir: dataDir})
}

// createCollection creates a collection or checks its existing configuration.
func createCollection(ctx context.Context, db core.VectorDB, name string, dimension int) error {
	config, err := db.GetCollection(ctx, name)
	if err == nil {
		if config.Dimension != dimension || config.Metric != model.MetricCosine {
			return fmt.Errorf("collection %q has dimension %d and metric %s; expected dimension %d and COSINE", name, config.Dimension, config.Metric, dimension)
		}
		return nil
	}
	var dbErr model.DBError
	if !errors.As(err, &dbErr) || dbErr.Code != model.ErrorCollectionNotFound {
		return err
	}
	return db.CreateCollection(ctx, model.CollectionConfig{Name: name, Dimension: dimension, Metric: model.MetricCosine, ShardCount: 1, ReplicationFactor: 1})
}

// addDocument embeds and stores a single document.
func addDocument(ctx context.Context, db core.VectorDB, client embeddingClient, collection string, doc document) error {
	return indexDocuments(ctx, db, client, collection, []document{doc})
}

// indexDocuments embeds all documents before writing the batch to the database.
func indexDocuments(ctx context.Context, db core.VectorDB, client embeddingClient, collection string, docs []document) error {
	if len(docs) == 0 {
		return nil
	}
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}
	vectors, err := client.embed(ctx, texts, client.documentPrefix)
	if err != nil {
		return err
	}
	points := make([]model.PointUpsert, len(docs))
	for i, doc := range docs {
		content := doc.Content
		points[i] = model.PointUpsert{ID: doc.ID, PartitionKey: doc.ID, Content: &content, Vector: &vectors[i]}
	}
	_, err = db.UpsertBatch(ctx, collection, points)
	return err
}

// search embeds a query and returns the closest documents.
func search(ctx context.Context, db core.VectorDB, client embeddingClient, collection, query string, topK int) ([]model.SearchResult, error) {
	vectors, err := client.embed(ctx, []string{query}, client.queryPrefix)
	if err != nil {
		return nil, err
	}
	return db.Search(ctx, collection, model.SearchQuery{Vector: vectors[0], TopK: topK})
}
