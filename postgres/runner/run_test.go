package postgresrunner_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Masterminds/squirrel"
	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"
	postgrescontainerrunner "github.com/amidgo/containers/postgres/runner"
	testmigrations "github.com/amidgo/containers/postgres/runner/testdata/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Test_Postgres_Migrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	migrations := goosemigrations.New(
		os.DirFS("./testdata/migrations"),
	)

	db := postgrescontainerrunner.RunForTesting(
		t,
		migrations,
		`INSERT INTO users (name) VALUES ('Dima')`,
		squirrel.Insert("users").Columns("name").Values("amidman").PlaceholderFormat(squirrel.Dollar),
	)

	assertUserExists(t, ctx, db, "Dima")
	assertUserExists(t, ctx, db, "amidman")
}

func Test_Postgres_EmbedMigrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db := postgrescontainerrunner.RunForTesting(
		t,
		goosemigrations.New(testmigrations.Embed()),
		`INSERT INTO users (name) VALUES ('Dima')`,
		squirrel.Insert("users").Columns("name").Values("amidman").PlaceholderFormat(squirrel.Dollar),
	)

	assertUserExists(t, ctx, db, "Dima")
	assertUserExists(t, ctx, db, "amidman")
}

func assertUserExists(t *testing.T, ctx context.Context, db *sql.DB, name string) {
	var userName string

	err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE name = $1", name).Scan(&userName)
	if err != nil {
		t.Errorf("assert user by %q name, %s", name, err)

		return
	}

	if userName != name {
		t.Errorf("assert user by %q name, wrong name %s", name, userName)
	}
}
