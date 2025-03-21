package goosemigrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"

	"github.com/amidgo/containers/postgres/migrations"
	"github.com/pressly/goose/v3"
)

type gooseMigrations struct {
	fs fs.FS
}

func New(folder string) migrations.Migrations {
	return gooseMigrations{
		fs: os.DirFS(folder),
	}
}

func Embed(fs fs.FS) migrations.Migrations {
	return gooseMigrations{
		fs: fs,
	}
}

func (g gooseMigrations) Up(ctx context.Context, db *sql.DB) error {
	gooseProvider, err := goose.NewProvider(goose.DialectPostgres, db, g.fs)
	if err != nil {
		return fmt.Errorf("create provider, %w", err)
	}

	report, err := gooseProvider.Up(ctx)
	if err != nil {
		return fmt.Errorf("up provider migrations, %w", err)
	}

	for _, r := range report {
		if r.Error == nil {
			continue
		}

		return err
	}

	return nil
}
