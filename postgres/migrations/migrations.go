package migrations

import (
	"context"
	"database/sql"
)

type Migrations interface {
	Up(ctx context.Context, db *sql.DB) error
}

var Nil nilMigrations

type nilMigrations struct{}

func (nilMigrations) Up(context.Context, *sql.DB) error {
	return nil
}
