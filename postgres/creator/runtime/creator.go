package postgresruntimecontainer

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	postgrescontainer "github.com/amidgo/containers/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var globalReusable = postgrescontainer.NewReusable(Default())

func GlobalReusable() *postgrescontainer.Reusable {
	return globalReusable
}

func Default() postgrescontainer.Creator {
	return New("test", "admin", "admin", "postgres:16-alpine", "pgx")
}

func New(
	dbName,
	dbUser,
	dbPassword,
	postgresImage,
	driverName string,
) postgrescontainer.Creator {
	return creator{
		dbName:        dbName,
		dbUser:        dbUser,
		dbPassword:    dbPassword,
		postgresImage: postgresImage,
		driverName:    driverName,
	}
}

type creator struct {
	dbName        string
	dbUser        string
	dbPassword    string
	postgresImage string
	driverName    string
}

func (c creator) Create(ctx context.Context) (postgrescontainer.Container, error) {
	dbName := c.dbName
	dbUser := c.dbUser
	dbPassword := c.dbPassword
	postgresImage := c.postgresImage
	driverName := c.driverName

	if img := os.Getenv("CONTAINERS_POSTGRES_IMAGE"); img != "" {
		postgresImage = img
	}

	postgresContainer, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("run container, %w", err)
	}

	cnt := container{
		driverName: driverName,
		cnt:        postgresContainer,
	}

	return cnt, nil
}

type container struct {
	driverName string
	cnt        *postgres.PostgresContainer
}

func (c container) Connect(ctx context.Context, args ...string) (*sql.DB, error) {
	dataSourceName, err := c.cnt.ConnectionString(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("get connection string, %w", err)
	}

	db, err := sql.Open(c.driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open connection, %w", err)
	}

	return db, nil
}

func (c container) Terminate(ctx context.Context) error {
	return c.cnt.Terminate(ctx)
}
