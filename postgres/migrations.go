package postgrescontainer

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

type Migrations interface {
	Up(db *sql.DB) error
	UpContext(ctx context.Context, db *sql.DB) error
}

type EmptyMigrations struct{}

func (EmptyMigrations) Up(*sql.DB) error {
	return nil
}

type gooseMigrations struct {
	folder string
}

func GooseMigrations(folder string) Migrations {
	return gooseMigrations{
		folder: folder,
	}
}

func (g gooseMigrations) UpContext(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, g.folder)
}

func (g gooseMigrations) Up(db *sql.DB) error {
	return goose.Up(db, g.folder)
}
