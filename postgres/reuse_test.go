package postgrescontainer_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Masterminds/squirrel"
	postgrescontainer "github.com/amidgo/containers/postgres"
	"github.com/amidgo/containers/postgres/migrations"
	goosemigrations "github.com/amidgo/containers/postgres/migrations/goose"
	postgrescontainerrunner "github.com/amidgo/containers/postgres/runner"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type containerTerminateWrapper struct {
	cnt  postgrescontainer.Container
	term func()
}

func (c containerTerminateWrapper) Connect(ctx context.Context, args ...string) (*sql.DB, error) {
	return c.cnt.Connect(ctx, args...)
}

func (c containerTerminateWrapper) Terminate(ctx context.Context) error {
	c.term()

	return c.cnt.Terminate(ctx)
}

func onTerminate(cnt postgrescontainer.Container, f func()) postgrescontainer.Container {
	return containerTerminateWrapper{
		cnt:  cnt,
		term: f,
	}
}

func Test_ReuseForTesting(t *testing.T) {
	t.Parallel()

	count := 0

	errCalledTwice := errors.New("called twice")

	testCcf := postgrescontainer.CreateContainerFunc(
		func(ctx context.Context) (postgrescontainer.Container, error) {
			if count != 0 {
				return nil, errCalledTwice
			}

			count++

			cnt, err := postgrescontainerrunner.RunContainer(nil)(ctx)

			cnt = onTerminate(cnt, func() {
				count--
			})

			return cnt, err
		},
	)

	testReusable := postgrescontainer.NewReusable(testCcf)

	t.Run("GlobalReuseable", testReuse(postgrescontainerrunner.Reusable()))
	t.Run("NewReuseable_RunContainer", testReuse(testReusable))
}

func Test_GooseMigrations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	db := postgrescontainer.ReuseForTesting(t,
		postgrescontainerrunner.Reusable(),
		migrations.Nil,
	)

	gooseMigrations := goosemigrations.New(
		os.DirFS(
			"./testdata/migrations",
		),
	)

	err := gooseMigrations.Up(ctx, db)
	if err != nil {
		t.Fatalf("up migrations: %s", err)
	}

	err = gooseMigrations.Down(ctx, db)
	if err != nil {
		t.Fatalf("down migrations: %s", err)
	}

	assertUsersTableDeleted(t, ctx, db)

	err = gooseMigrations.Up(ctx, db)
	if err != nil {
		t.Fatalf("up migrations after down: %s", err)
	}
}

func assertUsersTableDeleted(t *testing.T, ctx context.Context, db *sql.DB) {
	query := "SELECT * FROM users"

	_, err := db.ExecContext(ctx, query)

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		t.Fatalf("wrong error type, %+v", err)
	}

	if pgErr.Code != "42P01" {
		t.Fatalf("unexpected error code, expected %s, actual %s", "42P01", pgErr.Code)
	}
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
			goosemigrations.New(
				os.DirFS(
					"./testdata/migrations",
				),
			),
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
