package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/amidgo/containers"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	//nolint:revive // need for launch container
	_ "github.com/jackc/pgx/v5/stdlib"
)

func RunForTesting(t *testing.T, migrations Migrations, initialQueries ...string) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := RunContext(ctx, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start postgres container, err: %s", err)
	}

	return db
}

func Run(migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	return RunContext(context.Background(), migrations, initialQueries...)
}

func RunContext(ctx context.Context, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	return run(ctx, runContainer, migrations, initialQueries...)
}

func run(ctx context.Context, ccf createConatainerFunc, migrations Migrations, initialQueries ...string) (db *sql.DB, term func(), err error) {
	postgresContainer, err := runContainer(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	// Clean up the container
	term = func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	connString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, term, fmt.Errorf("get connection string, %w", err)
	}

	db, err = sql.Open("pgx", connString)
	if err != nil {
		return nil, term, fmt.Errorf("open connection, %w", err)
	}

	err = migrations.Up(db)
	if err != nil {
		return db, term, err
	}

	for _, initialQuery := range initialQueries {
		_, execErr := db.ExecContext(ctx, initialQuery)
		if execErr != nil {
			return db, term, fmt.Errorf("exec %s query, %w", initialQuery, execErr)
		}
	}

	return db, term, nil
}

func runContainer(ctx context.Context) (postgresContainer, error) {
	dbName := "test"
	dbUser := "admin"
	dbPassword := dbUser

	postgresImage := "postgres:16-alpine"

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

	return postgresContainer, nil
}
