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
	mig migrations.Migrations,
	initialQueries ...migrations.Query,
) (db *sql.DB, term func(), err error) {
	// Clean up the container
	term = func() {
		terminateErr := pgCnt.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("failed to terminate postgres container: %s", terminateErr)
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
			log.Printf("failed to terminate postgres container: %s", terminateErr)
		}
	}

	if mig != nil {
		err = mig.Up(ctx, db)
		if err != nil {
			return db, term, fmt.Errorf("up migrations, %w", err)
		}
	}

	for _, initialQuery := range initialQueries {
		err = migrations.ExecQuery(ctx, db, initialQuery)
		if err != nil {
			return db, term, err
		}
	}

	return db, term, nil
}
