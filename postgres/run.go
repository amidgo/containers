package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"

	"github.com/amidgo/containers"
	"github.com/amidgo/containers/postgres/migrations"
)

type Creator interface {
	Create(ctx context.Context) (Container, error)
}

func RunForTesting(
	t *testing.T,
	creator Creator,
	migrations migrations.Migrations,
	initialQueries ...string,
) *sql.DB {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db, term, err := Run(ctx, creator, migrations, initialQueries...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start postgres container, err: %s", err)
	}

	return db
}

func Run(
	ctx context.Context,
	creator Creator,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
	return run(ctx, creator, migrations, initialQueries...)
}

func run(
	ctx context.Context,
	creator Creator,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
	pgCnt, err := creator.Create(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	// Clean up the container
	term = func() {
		terminateErr := pgCnt.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("failed to terminate container: %s", terminateErr)
		}
	}

	db, err = pgCnt.Connect(ctx, "sslmode=disable")
	if err != nil {
		return nil, term, fmt.Errorf("connect to db, %w", err)
	}

	term = func() {
		_ = db.Close()

		terminateErr := pgCnt.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("failed to terminate container: %s", terminateErr)
		}
	}

	if migrations != nil {
		err = migrations.Up(ctx, db)
		if err != nil {
			return db, term, fmt.Errorf("up migrations, %w", err)
		}
	}

	for _, initialQuery := range initialQueries {
		_, execErr := db.ExecContext(ctx, initialQuery)
		if execErr != nil {
			return db, term, fmt.Errorf("exec %s query, %w", initialQuery, execErr)
		}
	}

	return db, term, nil
}
