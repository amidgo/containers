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

var (
	externalReusable = NewReusable(ExternalContainer(nil))
)

func ExternalReusable() *Reusable {
	return externalReusable
}

func ExternalForTestingConfig(
	t *testing.T,
	cfg *ExternalContainerConfig,
	migrations migrations.Migrations,
	initialQueries ...string,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := ExternalConfig(ctx, cfg, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatal(err)
	}

	return db
}

func ExternalForTesting(
	t *testing.T,
	migrations migrations.Migrations,
	initialQueries ...string,
) *sql.DB {
	return ExternalForTestingConfig(
		t,
		nil,
		migrations,
		initialQueries...,
	)
}

func ExternalConfig(
	ctx context.Context,
	cfg *ExternalContainerConfig,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
	pgCnt, err := ExternalContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	return Init(ctx, pgCnt, migrations, initialQueries...)
}

func External(
	ctx context.Context,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
	return ExternalConfig(
		ctx,
		nil,
		migrations,
		initialQueries...,
	)
}

type ExternalContainerConfig struct {
	DriverName       string
	ConnectionString string
}

var (
	defaultConfig = &ExternalContainerConfig{
		DriverName:       "pgx",
		ConnectionString: os.Getenv("CONTAINERS_POSTGRES_CONNECTION_STRING"),
	}
)

func ExternalContainer(cfg *ExternalContainerConfig) CreateContainerFunc {
	return func(context.Context) (Container, error) {
		if cfg == nil {
			cfg = defaultConfig
		}

		return externalContainer{
				connectionString: cfg.ConnectionString,
				driverName:       cfg.DriverName,
			},
			nil
	}
}

type externalContainer struct {
	connectionString string
	driverName       string
}

func (_ externalContainer) Terminate(ctx context.Context) error {
	return nil
}

func (e externalContainer) Connect(ctx context.Context, args ...string) (*sql.DB, error) {
	extraArgs := strings.Join(args, "&")

	dataSourceName := e.connectionString + "?" + extraArgs

	db, err := sql.Open(e.driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open connection to database, %w", err)
	}

	return db, nil
}
