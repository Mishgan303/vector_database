package service

import (
	"context"
	"sort"
	"strings"
	"sync"
	"vectordb/internal/model"
	"vectordb/internal/ports"
)

type collection struct {
	config model.CollectionConfig
	index  ports.VectorIndex
}
type CollectionRegistry struct {
	mu          sync.RWMutex
	collections map[string]*collection
}

// get returns a collection by name.
func (r *CollectionRegistry) get(name string) (*collection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collections[name]
	if !ok {
		return nil, model.DBError{Code: model.ErrorCollectionNotFound, Message: "collection not found"}
	}
	return c, nil
}

// list returns collection configurations sorted by name.
func (r *CollectionRegistry) list() []model.CollectionConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	configs := make([]model.CollectionConfig, 0, len(r.collections))
	for _, c := range r.collections {
		configs = append(configs, c.config)
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].Name < configs[j].Name })
	return configs
}

// validateConfig checks the collection configuration for the local database.
func validateConfig(config model.CollectionConfig) error {
	validMetric := config.Metric == model.MetricCosine || config.Metric == model.MetricDot || config.Metric == model.MetricL2
	if strings.TrimSpace(config.Name) == "" || config.Dimension <= 0 || !validMetric || config.ShardCount != 1 || config.ReplicationFactor != 1 {
		return model.DBError{Code: model.ErrorInvalidCollectionConfig, Message: "name, dimension, metric or local topology is invalid"}
	}
	return nil
}

// CreateCollection validates and creates a collection.
func (s *VectorDBService) CreateCollection(ctx context.Context, config model.CollectionConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.check(ctx); err != nil {
		return err
	}
	if err := validateConfig(config); err != nil {
		return err
	}
	if _, err := s.registry.get(config.Name); err == nil {
		return model.DBError{Code: model.ErrorCollectionAlreadyExists, Message: "collection already exists"}
	}
	op, err := makeOperation(model.WALCreateCollection, config.Name, config)
	if err != nil {
		return err
	}
	_, err = s.commit(ctx, op)
	return err
}

// DeleteCollection deletes a collection and its points.
func (s *VectorDBService) DeleteCollection(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.check(ctx); err != nil {
		return err
	}
	if _, err := s.registry.get(name); err != nil {
		return err
	}
	op, err := makeOperation(model.WALDeleteCollection, name, nil)
	if err != nil {
		return err
	}
	_, err = s.commit(ctx, op)
	return err
}

// GetCollection returns the configuration of a collection.
func (s *VectorDBService) GetCollection(ctx context.Context, name string) (model.CollectionConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.check(ctx); err != nil {
		return model.CollectionConfig{}, err
	}
	c, err := s.registry.get(name)
	if err != nil {
		return model.CollectionConfig{}, err
	}
	return c.config, nil
}

// ListCollections returns collection configurations sorted by name.
func (s *VectorDBService) ListCollections(ctx context.Context) ([]model.CollectionConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.check(ctx); err != nil {
		return nil, err
	}
	return s.registry.list(), nil
}
