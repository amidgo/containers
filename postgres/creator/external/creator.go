package posgresexternalcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	postgrescontainer "github.com/amidgo/containers/postgres"
)

var globalReusable = postgrescontainer.NewReusable(Default())

func GlobalReusable() *postgrescontainer.Reusable {
	return globalReusable
}

func Default() postgrescontainer.Creator {
	return New(
		envStringer{
			key: "CONTAINERS_POSTGRES_CONNECTION_STRING",
		},
		"pgx",
	)
}

func New(connectionString fmt.Stringer, driverName string) postgrescontainer.Creator {
	return Creator{
		connectionString: connectionString,
	}
}

type envStringer struct {
	key string
}

func (e envStringer) String() string {
	return os.Getenv(e.key)
}

type Creator struct {
	connectionString fmt.Stringer
	driverName       string
}

func (c Creator) Create(ctx context.Context) (postgrescontainer.Container, error) {
	return envContainer{
		connectionString: c.connectionString.String(),
		driverName:       c.driverName,
	}, nil
}

type envContainer struct {
	connectionString string
	driverName       string
}

func (_ envContainer) Terminate(ctx context.Context) error {
	return nil
}

func (e envContainer) Connect(ctx context.Context, args ...string) (*sql.DB, error) {
	extraArgs := strings.Join(args, "&")

	dataSourceName := e.connectionString + "?" + extraArgs

	db, err := sql.Open(e.driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open connection to database, %w", err)
	}

	return db, nil
}
