package miniocontainer

import (
	"context"

	"github.com/minio/minio-go/v7"
)

type Container interface {
	Connect(ctx context.Context) (*minio.Client, error)
	Terminate(ctx context.Context) error
}

type CreateContainerFunc func(ctx context.Context) (Container, error)
