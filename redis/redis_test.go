package containers_test

import (
	"context"
	"testing"

	rediscontainer "github.com/amidgo/containers/redis"
	"github.com/stretchr/testify/require"
)

func Test_RunRedis(t *testing.T) {
	t.Parallel()

	_ = rediscontainer.RunForTesting(t, nil)
}

func Test_RunRedis_Initial(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	redisClient := rediscontainer.RunForTesting(t, map[string]any{
		"key":     "value",
		"integer": 1000,
	})

	var (
		stringValue  string
		integerValue int
	)

	redisClient.Get(ctx, "key").Scan(&stringValue)
	redisClient.Get(ctx, "integer").Scan(&integerValue)

	require.Equal(t, "value", stringValue)
	require.Equal(t, 1000, integerValue)
}
