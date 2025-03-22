package miniocontainer

import (
	"context"
	"os"
	"testing"

	"github.com/amidgo/containers"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var externalReusable = NewReusable(ExternalContainer(nil))

func ExternalReusable() *Reusable {
	return externalReusable
}

func UseExternalForTestingConfig(
	t *testing.T,
	cfg *ExternalContainerConfig,
	buckets ...Bucket,
) *minio.Client {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	minioClient, term, err := UseExternalConfig(ctx, cfg, buckets...)
	t.Cleanup(term)

	if err != nil {
		t.Fatal(err)

		return nil
	}

	return minioClient
}

func UseExternalForTesting(t *testing.T, buckets ...Bucket) *minio.Client {
	return UseExternalForTestingConfig(t, nil, buckets...)
}

func UseExternalConfig(
	ctx context.Context,
	cfg *ExternalContainerConfig,
	buckets ...Bucket,
) (minioClient *minio.Client, term func(), err error) {
	cnt, err := ExternalContainer(cfg)(ctx)
	if err != nil {
		return nil, func() {}, err
	}

	return Init(ctx, cnt, buckets...)
}

func UseExternal(
	ctx context.Context,
	buckets ...Bucket,
) (minioClient *minio.Client, term func(), err error) {
	return UseExternalConfig(ctx, nil, buckets...)
}

type ExternalContainerConfig struct {
	Endpoint string
	User     string
	Password string
}

func externalContainerUser(cfg *ExternalContainerConfig) string {
	const defaultUser = "minio"

	if cfg != nil && cfg.User != "" {
		return cfg.User
	}

	return defaultUser
}

func externalContainerPassword(cfg *ExternalContainerConfig) string {
	const defaultPassword = "minio"

	if cfg != nil && cfg.Password != "" {
		return cfg.Password
	}

	return defaultPassword
}

func externalContainerEndpoint(cfg *ExternalContainerConfig) string {
	const endpointEnvName = "CONTAINERS_MINIO_ENDPOINT"

	if cfg != nil && cfg.Endpoint != "" {
		return cfg.Endpoint
	}

	envEndpoint := os.Getenv(endpointEnvName)
	if envEndpoint == "" {
		panic("endpoint is empty and environment variable " + endpointEnvName + " is empty")
	}

	return envEndpoint
}

func ExternalContainer(cfg *ExternalContainerConfig) CreateContainerFunc {
	return func(ctx context.Context) (Container, error) {
		endpoint := externalContainerEndpoint(cfg)
		user := externalContainerUser(cfg)
		password := externalContainerPassword(cfg)

		return externalContainer{
			endpoint: endpoint,
			userName: user,
			password: password,
		}, nil
	}
}

type externalContainer struct {
	endpoint string
	userName string
	password string
}

func (_ externalContainer) Terminate(context.Context) error {
	return nil
}

func (e externalContainer) Connect(ctx context.Context) (*minio.Client, error) {
	return minio.New(e.endpoint,
		&minio.Options{
			Creds:           credentials.NewStaticV4(e.userName, e.password, ""),
			TrailingHeaders: true,
		},
	)
}
