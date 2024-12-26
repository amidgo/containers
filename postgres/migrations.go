package postgrescontainer

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

type Migrations interface {
	Up(*sql.DB) error
	Down(*sql.DB) error
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

func (g gooseMigrations) Up(db *sql.DB) error {
	return goose.Up(db, g.folder)
}

func (g gooseMigrations) Down(db *sql.DB) error {
	return goose.Down(db, g.folder)
}
