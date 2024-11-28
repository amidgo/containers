package miniocontainer

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"errors"

	"github.com/amidgo/containers"
	minioclient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

type File struct {
	Name    string
	Bucket  string
	Content string
}

func RunForTesting(t *testing.T, availableBuckets []string, initialFiles []File) *minioclient.Client {
	containers.SkipDisabled(t)

	minioClient, term, err := Run(availableBuckets, initialFiles)
	t.Cleanup(term)

	if err != nil {
		t.Fatalf("start minio container, err: %s", err)
	}

	return minioClient
}

type PutFileError struct {
	Name   string
	Bucket string
}

func (p PutFileError) Error() string {
	return fmt.Sprintf("put file %s in %s", p.Name, p.Bucket)
}

func Run(availableBuckets []string, initialFiles []File) (client *minioclient.Client, term func(), err error) {
	ctx := context.Background()

	minioImage := "minio/minio:RELEASE.2024-01-16T16-07-38Z"

	if img := os.Getenv("CONTAINERS_MINIO_IMAGE"); img != "" {
		minioImage = img
	}

	minioContainer, err := minio.Run(ctx, minioImage)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	// Clean up the container
	term = func() {
		if err := minioContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	endpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		return nil, term, fmt.Errorf("get connection string, %w", err)
	}

	minioClient, err := minioclient.New(endpoint,
		&minioclient.Options{
			Creds:           credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
			TrailingHeaders: true,
		},
	)
	if err != nil {
		return nil, term, fmt.Errorf("create minio client, %w", err)
	}

	for _, bucket := range availableBuckets {
		err := minioClient.MakeBucket(ctx, bucket, minioclient.MakeBucketOptions{})
		if err != nil {
			return nil, term, fmt.Errorf("create bucket %s, %w", bucket, err)
		}
	}

	for _, file := range initialFiles {
		exists, err := minioClient.BucketExists(ctx, file.Bucket)
		if err != nil {
			return nil, term, fmt.Errorf("get bucket exits %s, %w", file.Bucket, err)
		}

		if !exists {
			err := minioClient.MakeBucket(ctx, file.Bucket, minioclient.MakeBucketOptions{})
			if err != nil {
				return nil, term, fmt.Errorf("create bucket %s, %w", file.Bucket, err)
			}
		}

		_, err = minioClient.PutObject(ctx,
			file.Bucket,
			file.Name,
			strings.NewReader(file.Content),
			-1,
			minioclient.PutObjectOptions{},
		)
		if err != nil {
			return nil, term, errors.Join(PutFileError{Name: file.Name, Bucket: file.Bucket}, err)
		}
	}

	return minioClient, term, nil
}
