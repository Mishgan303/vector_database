package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type embeddingClient struct {
	baseURL        string
	model          string
	documentPrefix string
	queryPrefix    string
	httpClient     *http.Client
}

// embed requests one vector for each text from the selected Ollama model.
func (c embeddingClient) embed(ctx context.Context, texts []string, prefix string) ([][]float32, error) {
	inputs := make([]string, len(texts))
	for i, text := range texts {
		inputs[i] = prefix + text
	}
	body, err := json.Marshal(struct {
		Model    string   `json:"model"`
		Input    []string `json:"input"`
		Truncate bool     `json:"truncate"`
	}{Model: c.model, Input: inputs, Truncate: false})
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request embeddings: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Ollama returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}
	if len(result.Embeddings) > 0 {
		dimension := len(result.Embeddings[0])
		if dimension == 0 {
			return nil, fmt.Errorf("Ollama returned an empty embedding")
		}
		for _, vector := range result.Embeddings {
			if len(vector) != dimension {
				return nil, fmt.Errorf("Ollama returned embeddings with different dimensions")
			}
		}
	}
	return result.Embeddings, nil
}
