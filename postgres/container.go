package postgrescontainer

import "context"

type postgresContainer interface {
	ConnectionString(ctx context.Context, args ...string) (string, error)
	Terminate(ctx context.Context) error
}

type createConatainerFunc func(ctx context.Context) (postgresContainer, error)
