package goosemigrations

import (
	"context"
	"database/sql"

	"github.com/amidgo/containers/postgres/migrations"
	"github.com/pressly/goose/v3"
)

type gooseMigrations struct {
	folder string
}

func New(folder string) migrations.Migrations {
	return gooseMigrations{
		folder: folder,
	}
}

func (g gooseMigrations) Up(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, g.folder)
}
