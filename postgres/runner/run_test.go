package postgrescontainerrunner_test

import (
	"context"
	"testing"

	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"
	postgrescontainerrunner "github.com/amidgo/containers/postgres/runner"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Test_Postgres_Migrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	db := postgrescontainerrunner.RunForTesting(
		t,
		goosemigrations.New("./testdata/migrations"),
		`INSERT INTO users (name) VALUES ('Dima')`,
	)

	expectedName := "Dima"

	name := ""

	err := db.QueryRowContext(context.Background(), "SELECT name FROM users").Scan(&name)
	if err != nil {
		t.Fatalf("select name from users, unexpected error: %+v", err)
	}

	if expectedName != name {
		t.Fatalf("wrong name, expected %s, actual %s", expectedName, name)
	}
}
