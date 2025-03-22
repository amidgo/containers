package miniorunner

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/amidgo/containers"
	miniocontainer "github.com/amidgo/containers/minio"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	miniocnt "github.com/testcontainers/testcontainers-go/modules/minio"
)

func RunForTesting(
	t *testing.T,
	buckets ...miniocontainer.Bucket,
) *minio.Client {
	return RunForTestingConfig(t, nil, buckets...)
}

func RunForTestingConfig(
	t *testing.T,
	cfg *ContainerConfig,
	buckets ...miniocontainer.Bucket,
) *minio.Client {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	minioClient, term, err := RunConfig(ctx, cfg, buckets...)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("run minio container with config, %s", err.Error())

		return nil
	}

	return minioClient
}

func Run(
	ctx context.Context,
	buckets ...miniocontainer.Bucket,
) (minioClient *minio.Client, term func(), err error) {
	return RunConfig(ctx, nil, buckets...)
}

func RunConfig(
	ctx context.Context,
	cfg *ContainerConfig,
	buckets ...miniocontainer.Bucket,
) (minioClient *minio.Client, term func(), err error) {
	cnt, err := RunContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("run container, %w", err)
	}

	return miniocontainer.Init(ctx, cnt, buckets...)
}

type ContainerConfig struct {
	MinioImage string
	Username   string
	Password   string
}

func containerMinioImage(cfg *ContainerConfig) string {
	const defaultMinioImage = "minio/minio:RELEASE.2024-01-16T16-07-38Z"

	if cfg != nil && cfg.MinioImage != "" {
		return cfg.MinioImage
	}

	envContainerImage := os.Getenv("CONTAINERS_MINIO_IMAGE")
	if envContainerImage != "" {
		return envContainerImage
	}

	return defaultMinioImage
}

func containerUsername(cfg *ContainerConfig) string {
	const defaultUsername = "minioadmin"

	if cfg != nil && cfg.Username != "" {
		return cfg.Username
	}

	return defaultUsername
}

func containerPassword(cfg *ContainerConfig) string {
	const defaultPassword = "minioadmin"

	if cfg != nil && cfg.Password != "" {
		return cfg.Password
	}

	return defaultPassword
}

func RunContainer(cfg *ContainerConfig) miniocontainer.CreateContainerFunc {
	return func(ctx context.Context) (miniocontainer.Container, error) {
		minioImage := containerMinioImage(cfg)
		username := containerUsername(cfg)
		password := containerPassword(cfg)

		cnt, err := miniocnt.Run(ctx,
			minioImage,
			miniocnt.WithUsername(username),
			miniocnt.WithPassword(password),
		)
		if err != nil {
			return nil, fmt.Errorf("run minio container, %w", err)
		}

		return container{
			minioContainer: cnt,
		}, nil
	}
}

type container struct {
	minioContainer *miniocnt.MinioContainer
}

func (c container) Connect(ctx context.Context) (*minio.Client, error) {
	endpoint, err := c.minioContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to minio container, get endpoint, %w", err)
	}

	opts := &minio.Options{
		Creds: credentials.NewStaticV4(c.minioContainer.Username, c.minioContainer.Password, ""),
	}

	minioClient, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("create minio client, %w", err)
	}

	return minioClient, nil
}

func (c container) Terminate(ctx context.Context) error {
	return c.minioContainer.Terminate(ctx)
}
