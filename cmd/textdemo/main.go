package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// loadDocuments reads each text file as one document and uses its filename as the ID.
func loadDocuments(dir string) ([]document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var docs []document
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		docs = append(docs, document{ID: strings.TrimSuffix(entry.Name(), ".txt"), Content: strings.TrimSpace(string(content))})
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no .txt documents found in %s", dir)
	}
	return docs, nil
}

func main() {
	modelName := flag.String("model", "", "Ollama embedding model name (required)")
	baseURL := flag.String("ollama", "http://localhost:11434", "Ollama base URL")
	dataDir := flag.String("data", "./data/textdemo", "database data directory")
	collection := flag.String("collection", "library", "collection name")
	dataset := flag.String("dataset", "./datasets/rag/library", "directory of .txt documents")
	query := flag.String("query", "Как продлить книгу?", "search query")
	topK := flag.Int("top-k", 3, "maximum number of results")
	docPrefix := flag.String("document-prefix", "", "prefix added to document text for the model")
	queryPrefix := flag.String("query-prefix", "", "prefix added to search queries for the model")
	flag.Parse()
	if strings.TrimSpace(*modelName) == "" {
		fmt.Fprintln(os.Stderr, "specify an Ollama embedding model with -model")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := embeddingClient{baseURL: *baseURL, model: *modelName, documentPrefix: *docPrefix, queryPrefix: *queryPrefix, httpClient: &http.Client{Timeout: 2 * time.Minute}}
	if err := run(ctx, client, *dataDir, *collection, *dataset, *query, *topK); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run loads documents, indexes them, and searches the local database.
func run(ctx context.Context, client embeddingClient, dataDir, collection, dataset, query string, topK int) (err error) {
	docs, err := loadDocuments(dataset)
	if err != nil {
		return fmt.Errorf("load dataset: %w", err)
	}
	probe, err := client.embed(ctx, []string{docs[0].Content}, client.documentPrefix)
	if err != nil {
		return err
	}
	db, err := initializeDB(ctx, dataDir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close(context.Background())) }()
	if err = createCollection(ctx, db, collection, len(probe[0])); err != nil {
		return err
	}
	if err = indexDocuments(ctx, db, client, collection, docs); err != nil {
		return err
	}
	results, err := search(ctx, db, client, collection, query, topK)
	if err != nil {
		return err
	}
	fmt.Printf("Query: %s\nIndexed: %d documents\n\n", query, len(docs))
	for _, result := range results {
		fmt.Printf("ID: %s | Score: %.4f\n%s\n\n", result.Ref.ID, result.Score, result.Content)
	}
	return nil
}
