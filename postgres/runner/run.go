package pgrunner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/amidgo/containers"
	postgrescontainer "github.com/amidgo/containers/postgres"
	"github.com/amidgo/containers/postgres/migrations"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func RunForTestingConfig(
	t *testing.T,
	cfg *Config,
	migrations migrations.Migrations,
	initialQueries ...postgrescontainer.Query,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := RunConfig(ctx, cfg, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatal(err)
	}

	return db
}

func RunForTesting(
	t *testing.T,
	migrations migrations.Migrations,
	initialQueries ...postgrescontainer.Query,
) *sql.DB {
	return RunForTestingConfig(
		t,
		nil,
		migrations,
		initialQueries...,
	)
}

func Run(
	ctx context.Context,
	migrations migrations.Migrations,
	initialQueries ...postgrescontainer.Query,
) (db *sql.DB, term func(), err error) {
	return RunConfig(ctx, nil, migrations, initialQueries...)
}

func RunConfig(
	ctx context.Context,
	cfg *Config,
	migrations migrations.Migrations,
	initialQueries ...postgrescontainer.Query,
) (db *sql.DB, term func(), err error) {
	pgCnt, err := RunContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	return postgrescontainer.Init(ctx, pgCnt, migrations, initialQueries...)
}

type Config struct {
	DBName        string
	DBUser        string
	DBPassword    string
	PostgresImage string
	DriverName    string
}

const (
	defaultDBName        = "test"
	defaultDBUser        = "admin"
	defaultDBPassword    = "admin"
	defaultPostgresImage = "postgres:16-alpine"
	defaultDriverName    = "pgx"
)

var defaultConfig = &Config{
	DBName:        defaultDBName,
	DBUser:        defaultDBUser,
	DBPassword:    defaultDBPassword,
	PostgresImage: defaultPostgresImage,
	DriverName:    defaultDriverName,
}

func RunContainer(cfg *Config) postgrescontainer.CreateContainerFunc {
	return func(ctx context.Context) (postgrescontainer.Container, error) {
		if cfg == nil {
			cfg = defaultConfig
		}

		dbName := cfg.DBName
		dbUser := cfg.DBUser
		dbPassword := cfg.DBPassword
		postgresImage := os.Getenv("CONTAINERS_POSTGRES_IMAGE")
		driverName := cfg.DriverName

		if dbName == "" {
			dbName = defaultDBName
		}

		if dbUser == "" {
			dbUser = defaultDBUser
		}

		if dbPassword == "" {
			dbPassword = defaultDBPassword
		}

		if postgresImage == "" {
			postgresImage = defaultPostgresImage
		}

		if driverName == "" {
			driverName = defaultDriverName
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
