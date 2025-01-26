package postgrescontainer_test

import (
	"fmt"
	"testing"

	postgrescontainer "github.com/amidgo/containers/postgres"
)

func Test_ReuseForTesting(t *testing.T) {
	t.Parallel()

	t.Run("GlobalReuseable", testReuse(postgrescontainer.GlobalReusable()))
	t.Run("NewReuseable_RunContainer", testReuse(postgrescontainer.NewReusable(postgrescontainer.RunContainer)))
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
			postgrescontainer.GooseMigrations("./testdata/migrations"),
			"INSERT INTO users (name) VALUES ('Dima')",
		)
	}
}
