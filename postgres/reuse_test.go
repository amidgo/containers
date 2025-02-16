package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/Masterminds/squirrel"
	postgrescontainer "github.com/amidgo/containers/postgres"
	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"
	postgrescontainerrunner "github.com/amidgo/containers/postgres/runner"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Test_ReuseForTesting(t *testing.T) {
	t.Parallel()

	testReusable := postgrescontainer.NewReusable(
		postgrescontainerrunner.RunContainer(nil),
	)

	t.Run("GlobalReuseable", testReuse(postgrescontainerrunner.Reusable()))
	t.Run("NewReuseable_RunContainer", testReuse(testReusable))
}

func testReuse(reusable *postgrescontainer.Reusable) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		reuseCaseCount := 10000

		if testing.Short() {
			reuseCaseCount = 100
		}

		for i := range reuseCaseCount {
			t.Run(fmt.Sprintf("%d", i), runReuseCase(reusable))
		}
	}
}

func runReuseCase(reusable *postgrescontainer.Reusable) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		db := postgrescontainer.ReuseForTesting(t,
			reusable,
			goosemigrations.New("./testdata/migrations"),
			"INSERT INTO users (name) VALUES ('Dima')",
			squirrel.Insert("users").Columns("name").Values("amidman").PlaceholderFormat(squirrel.Dollar),
		)

		assertUserExists(t, ctx, db, "Dima")
		assertUserExists(t, ctx, db, "amidman")
	}
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
