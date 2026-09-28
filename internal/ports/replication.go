package ports

import "context"

type Replicator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
