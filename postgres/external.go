package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/amidgo/containers"
	"github.com/amidgo/containers/postgres/migrations"
)

var externalReusable = NewReusable(ExternalContainer(nil))

func ExternalReusable() *Reusable {
	return externalReusable
}

func UseExternalForTestingConfig(
	t *testing.T,
	cfg *ExternalContainerConfig,
	migrations migrations.Migrations,
	initialQueries ...Query,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := UseExternalConfig(ctx, cfg, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatal(err)

		return nil
	}

	return db
}

func UseExternalForTesting(
	t *testing.T,
	migrations migrations.Migrations,
	initialQueries ...Query,
) *sql.DB {
	return UseExternalForTestingConfig(
		t,
		nil,
		migrations,
		initialQueries...,
	)
}

func UseExternalConfig(
	ctx context.Context,
	cfg *ExternalContainerConfig,
	migrations migrations.Migrations,
	initialQueries ...Query,
) (db *sql.DB, term func(), err error) {
	pgCnt, err := ExternalContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	return Init(ctx, pgCnt, migrations, initialQueries...)
}

func UseExternal(
	ctx context.Context,
	migrations migrations.Migrations,
	initialQueries ...Query,
) (db *sql.DB, term func(), err error) {
	var cfg *ExternalContainerConfig

	return UseExternalConfig(
		ctx,
		cfg,
		migrations,
		initialQueries...,
	)
}

type ExternalContainerConfig struct {
	DriverName       string
	ConnectionString string
}

func externalContainerDriverName(cfg *ExternalContainerConfig) string {
	const defaultDriverName = "pgx"

	if cfg != nil && cfg.DriverName != "" {
		return cfg.DriverName
	}

	return defaultDriverName
}

func externalContainerConnectionString(cfg *ExternalContainerConfig) string {
	const connectionStringEnvName = "CONTAINERS_POSTGRES_CONNECTION_STRING"

	if cfg != nil && cfg.ConnectionString != "" {
		return cfg.ConnectionString
	}

	defaultConnectionString := os.Getenv(connectionStringEnvName)

	if defaultConnectionString == "" {
		panic("connection string is empty and environment variable " + connectionStringEnvName + " is empty")
	}

	return defaultConnectionString
}

func ExternalContainer(cfg *ExternalContainerConfig) CreateContainerFunc {
	return func(context.Context) (Container, error) {
		connectionString := externalContainerConnectionString(cfg)
		driverName := externalContainerDriverName(cfg)

		return externalContainer{
				connectionString: connectionString,
				driverName:       driverName,
			},
			nil
	}
}

type externalContainer struct {
	connectionString string
	driverName       string
}

func (_ externalContainer) Terminate(_ context.Context) error {
	return nil
}

func (e externalContainer) Connect(_ context.Context, args ...string) (*sql.DB, error) {
	extraArgs := strings.Join(args, "&")

	dataSourceName := e.connectionString + "?" + extraArgs

	db, err := sql.Open(e.driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open connection to database, %w", err)
	}

	return db, nil
}
