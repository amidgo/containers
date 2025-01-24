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

func testReuse(reuseable *postgrescontainer.Reuseable) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		for i := range 100 {
			t.Run(fmt.Sprintf("%d", i), runReuseCase(reuseable))
		}
	}
}

func runReuseCase(reuseable *postgrescontainer.Reuseable) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		_ = postgrescontainer.ReuseForTesting(t,
			reuseable,
			postgrescontainer.GooseMigrations("./testdata/migrations"),
			"INSERT INTO users (name) VALUES ('Dima')",
		)
	}
}
