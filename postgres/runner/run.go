package postgresrunner

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
	cfg *ContainerConfig,
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

		return nil
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
	cfg *ContainerConfig,
	migrations migrations.Migrations,
	initialQueries ...postgrescontainer.Query,
) (db *sql.DB, term func(), err error) {
	pgCnt, err := RunContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	return postgrescontainer.Init(ctx, pgCnt, migrations, initialQueries...)
}

type ContainerConfig struct {
	DBName                    string
	DBUser                    string
	DBPassword                string
	PostgresImage             string
	DriverName                string
	DisableTestContainersLogs bool
}

func containerDBName(cfg *ContainerConfig) string {
	const defaultDBName = "test"

	if cfg != nil && cfg.DBName != "" {
		return cfg.DBName
	}

	return defaultDBName
}

func containerDBUser(cfg *ContainerConfig) string {
	const defaultDBUser = "admin"

	if cfg != nil && cfg.DBUser != "" {
		return cfg.DBUser
	}

	return defaultDBUser
}

func containerDBPassword(cfg *ContainerConfig) string {
	const defaultDBPassword = "admin"

	if cfg != nil && cfg.DBPassword != "" {
		return cfg.DBPassword
	}

	return defaultDBPassword
}

func containerPostgresImage(cfg *ContainerConfig) string {
	const defaultPostgresImage = "postgres:16-alpine"

	if cfg != nil && cfg.PostgresImage != "" {
		return cfg.PostgresImage
	}

	envPostgresImage := os.Getenv("CONTAINERS_POSTGRES_IMAGE")
	if envPostgresImage != "" {
		return envPostgresImage
	}

	return defaultPostgresImage
}

func containerDriverName(cfg *ContainerConfig) string {
	const defaultDriverName = "pgx"

	if cfg != nil && cfg.DriverName != "" {
		return cfg.DriverName
	}

	return defaultDriverName
}

func containerDisableTestContainersLogs(cfg *ContainerConfig) bool {
	if cfg == nil {
		return false
	}

	return cfg.DisableTestContainersLogs
}

func RunContainer(cfg *ContainerConfig) postgrescontainer.CreateContainerFunc {
	return func(ctx context.Context) (postgrescontainer.Container, error) {
		postgresImage := containerPostgresImage(cfg)
		dbName := containerDBName(cfg)
		dbUser := containerDBUser(cfg)
		dbPassword := containerDBPassword(cfg)

		opts := []testcontainers.ContainerCustomizer{
			postgres.WithDatabase(dbName),
			postgres.WithUsername(dbUser),
			postgres.WithPassword(dbPassword),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
			),
		}

		if containerDisableTestContainersLogs(cfg) {
			opts = append(opts, testcontainers.WithLogger(noopLogger{}))
		}

		postgresContainer, err := postgres.Run(ctx,
			postgresImage,
			opts...,
		)
		if err != nil {
			return nil, fmt.Errorf("run container, %w", err)
		}

		driverName := containerDriverName(cfg)

		cnt := container{
			driverName: driverName,
			cnt:        postgresContainer,
		}

		return cnt, nil
	}

}

type noopLogger struct{}

func (noopLogger) Printf(string, ...interface{}) {}

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
