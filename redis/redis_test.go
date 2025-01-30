package containers_test

import (
	"context"
	"testing"

	rediscontainer "github.com/amidgo/containers/redis"
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

	if stringValue != "value" {
		t.Fatalf("unexpected value from stringValue, expected 'value', actual %s", stringValue)
	}

	if integerValue != 1000 {
		t.Fatalf("unexpected value from integerValue, expected 1000, actual %d", integerValue)
	}
}
