package postgrescontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/amidgo/containers/postgres/migrations"
)

func Init(
	ctx context.Context,
	pgCnt Container,
	migrations migrations.Migrations,
	initialQueries ...string,
) (db *sql.DB, term func(), err error) {
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
