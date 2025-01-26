package postgrescontainer

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

type Migrations interface {
	Up(ctx context.Context, db *sql.DB) error
}

type gooseMigrations struct {
	folder string
}

func GooseMigrations(folder string) Migrations {
	return gooseMigrations{
		folder: folder,
	}
}

func (g gooseMigrations) Up(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, g.folder)
}

var EmptyMigrations Migrations = emptyMigrations{}

type emptyMigrations struct{}

func (emptyMigrations) Up(context.Context, *sql.DB) error {
	return nil
}
