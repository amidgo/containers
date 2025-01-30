package postgrescontainer_test

import (
	"context"
	"testing"

	postgrescontainer "github.com/amidgo/containers/postgres"
	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"
)

func Test_Postgres_Migrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	db := postgrescontainer.RunForTesting(
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
