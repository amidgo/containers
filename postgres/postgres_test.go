package postgrescontainer_test

import (
	"context"
	"testing"

	postgrescontainer "github.com/amidgo/containers/postgres"
	"github.com/stretchr/testify/require"
)

func Test_Postgres_Migrations_WithInitialQuery(t *testing.T) {
	t.Parallel()

	db := postgrescontainer.RunForTesting(
		t,
		postgrescontainer.GooseMigrations("./testdata/migrations"),
		`INSERT INTO users (name) VALUES ('Dima')`,
	)

	expectedName := "Dima"

	name := ""

	err := db.QueryRowContext(context.Background(), "SELECT name FROM users").Scan(&name)
	require.NoError(t, err)

	require.Equal(t, expectedName, name)
}
