package goosemigrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/amidgo/containers/postgres/migrations"
	"github.com/pressly/goose/v3"
)

type gooseMigrations struct {
	fsys fs.FS
}

func New(fsys fs.FS) migrations.Migrations {
	return gooseMigrations{
		fsys: fsys,
	}
}

func (g gooseMigrations) Up(ctx context.Context, db *sql.DB) error {
	gooseProvider, err := goose.NewProvider(goose.DialectPostgres, db, g.fsys)
	if err != nil {
		return fmt.Errorf("create provider, %w", err)
	}

	report, err := gooseProvider.Up(ctx)
	if err != nil {
		return fmt.Errorf("up migrations, %w", err)
	}

	for _, r := range report {
		if r.Error == nil {
			continue
		}

		return err
	}

	return nil
}

func (g gooseMigrations) Down(ctx context.Context, db *sql.DB) error {
	gooseProvider, err := goose.NewProvider(goose.DialectPostgres, db, g.fsys)
	if err != nil {
		return fmt.Errorf("create provider, %w", err)
	}

	report, err := gooseProvider.DownTo(ctx, 0)
	if err != nil {
		return fmt.Errorf("down migrations, %w", err)
	}

	for _, r := range report {
		if r.Error == nil {
			continue
		}

		return err
	}

	return nil
}
