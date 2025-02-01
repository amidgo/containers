package postgrescontainer_test

import (
	"fmt"
	"testing"

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

		_ = postgrescontainer.ReuseForTesting(t,
			reusable,
			goosemigrations.New("./testdata/migrations"),
			"INSERT INTO users (name) VALUES ('Dima')",
		)
	}
}
