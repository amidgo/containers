package postgrescontainer_test

import (
	"context"
	"testing"

	postgrescontainer "github.com/amidgo/containers/postgres"
	postgresruntimecontainer "github.com/amidgo/containers/postgres/creator/runtime"
	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Test_Postgres_Migrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	db := postgrescontainer.RunForTesting(
		t,
		postgresruntimecontainer.Default(),
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
