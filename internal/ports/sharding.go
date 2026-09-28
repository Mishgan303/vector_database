package ports

import "vectordb/internal/model"

type ShardRouter interface {
	ShardFor(partitionKey string) (model.ShardID, error)
	AllShards() []model.ShardID
}
