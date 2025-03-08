package postgrescontainer

import (
	"context"
	"database/sql"
)

type Container interface {
	Connect(ctx context.Context, args ...string) (*sql.DB, error)
	Terminate(ctx context.Context) error
}

type CreateContainerFunc func(ctx context.Context) (Container, error)
