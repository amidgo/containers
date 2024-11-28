package containers

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	redis "github.com/redis/go-redis/v9"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func RunForTesting(t *testing.T, initial map[string]any) *redis.Client {
	redisClient, term, err := Run(initial)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start redis container, err: %s", err)
	}

	return redisClient
}

func Run(initial map[string]any) (redisClient *redis.Client, term func(), err error) {
	ctx := context.Background()

	redisImage := "redis:6"

	if img := os.Getenv("CONTAINERS_REDIS_IMAGE"); img != "" {
		redisImage = img
	}

	redisContainer, err := rediscontainer.Run(ctx, redisImage)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	term = func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	addr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		return nil, term, fmt.Errorf("get connection string, %w", err)
	}

	opts, err := redis.ParseURL(addr)
	if err != nil {
		return nil, term, fmt.Errorf("parse url: %s, %w", addr, err)
	}

	client := redis.NewClient(opts)

	err = client.Ping(ctx).Err()
	if err != nil {
		return nil, term, fmt.Errorf("ping client, %w", err)
	}

	if initial != nil {
		err := initializeRedis(ctx, client, initial)
		if err != nil {
			return nil, term, fmt.Errorf("initialize redis, %w", err)
		}
	}

	return client, term, nil
}

func initializeRedis(ctx context.Context, client *redis.Client, initial map[string]any) error {
	for key, value := range initial {
		err := client.Set(ctx, key, value, 0).Err()
		if err != nil {
			return fmt.Errorf("set, key: %s, value: %s, %w", key, value, err)
		}
	}

	return nil
}
