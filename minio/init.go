package miniocontainer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	minioclient "github.com/minio/minio-go/v7"
)

type Bucket struct {
	Name  string
	Files []File
}

type File struct {
	Name    string
	Content string
}

func Init(
	ctx context.Context,
	cnt Container,
	buckets ...Bucket,
) (minioClient *minio.Client, term func(), err error) {
	term = func() {
		terminateErr := cnt.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("failed to terminate minio container: %s", terminateErr)
		}
	}

	minioClient, err = cnt.Connect(ctx)
	if err != nil {
		return nil, term, fmt.Errorf("connect to minio container, %w", err)
	}

	err = insertBuckets(ctx, minioClient, buckets...)
	if err != nil {
		return nil, term, err
	}

	return minioClient, term, nil
}

func insertBuckets(ctx context.Context, minioClient *minio.Client, buckets ...Bucket) error {
	for _, bucket := range buckets {
		err := insertSingleBucket(ctx, minioClient, bucket)
		if err != nil {
			return err
		}
	}

	return nil
}

func insertSingleBucket(ctx context.Context, minioClient *minio.Client, bucket Bucket) error {
	bucketExists, err := minioClient.BucketExists(ctx, bucket.Name)
	if err != nil {
		return fmt.Errorf("get bucket exits %s, %w", bucket.Name, err)
	}

	if !bucketExists {
		makeBucketOpts := minioclient.MakeBucketOptions{}

		err := minioClient.MakeBucket(ctx, bucket.Name, makeBucketOpts)
		if err != nil {
			return fmt.Errorf("create bucket %s, %w", bucket.Name, err)
		}
	}

	putObjectOpts := minioclient.PutObjectOptions{}

	for _, file := range bucket.Files {
		_, err = minioClient.PutObject(ctx,
			bucket.Name,
			file.Name,
			strings.NewReader(file.Content),
			-1,
			putObjectOpts,
		)
		if err != nil {
			return fmt.Errorf("put file %s into bucket %s, %w", file.Name, bucket.Name, err)
		}
	}

	return nil
}
