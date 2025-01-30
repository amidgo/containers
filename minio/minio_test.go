package miniocontainer_test

import (
	"context"
	"io"
	"strings"
	"testing"

	miniocontainer "github.com/amidgo/containers/minio"
	"github.com/minio/minio-go/v7"
)

type RunForTestingTest struct {
	CaseName         string
	AvailableBuckets []string
	InitialFiles     []miniocontainer.File
}

func (r *RunForTestingTest) Name() string {
	return r.CaseName
}

func (r *RunForTestingTest) Test(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	minioClient := miniocontainer.RunForTesting(t, r.AvailableBuckets, r.InitialFiles)

	_, err := minioClient.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("get list of buckets, unexpected error: %+v", err)
	}

	for _, bucket := range r.AvailableBuckets {
		exists, err := minioClient.BucketExists(ctx, bucket)
		if err != nil {
			t.Fatalf("check bucket %s exists, unexpected error: %+v", bucket, err)
		}

		if !exists {
			t.Fatalf("check bucket %s exists, bucket not exists", bucket)
		}
	}

	for _, initialFile := range r.InitialFiles {
		requireInitialFileExists(t, ctx, minioClient, initialFile)
	}
}

func requireInitialFileExists(t *testing.T, ctx context.Context, minioClient *minio.Client, initialFile miniocontainer.File) {
	object, err := minioClient.GetObject(ctx, initialFile.Bucket, initialFile.Name, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("get object %s from bucket %s, unexpected error: %+v", initialFile.Name, initialFile.Bucket, err)
	}

	objectData := &strings.Builder{}

	_, err = io.Copy(objectData, object)
	if err != nil {
		t.Fatalf("read data from object %s from bucket %s, unexpected error: %+v", initialFile.Name, initialFile.Bucket, err)
	}

	if objectData.String() != initialFile.Content {
		t.Fatalf(
			"objectData from %s from bucket %s not equal,\nexpected:\n\t%s\nactual:\n\t%s",
			initialFile.Name,
			initialFile.Bucket,
			initialFile.Content,
			objectData.String(),
		)
	}
}

func runForTestingTests(
	t *testing.T,
	tests ...*RunForTestingTest,
) {
	for _, tst := range tests {
		t.Run(tst.Name(), tst.Test)
	}
}

func Test_Minio(t *testing.T) {
	t.Parallel()

	runForTestingTests(t,
		&RunForTestingTest{
			CaseName: "empty buckets and files",
		},
		&RunForTestingTest{
			CaseName: "only buckets filled",
			AvailableBuckets: []string{
				"first",
				"second",
				"third",
			},
		},
		&RunForTestingTest{
			CaseName: "all filled, no conflicts",
			AvailableBuckets: []string{
				"first",
				"second",
				"third",
			},
			InitialFiles: []miniocontainer.File{
				{
					Name:    "Gagarin.pdf",
					Bucket:  "first",
					Content: "Поехали!",
				},
				{
					Name:    "Titov.pdf",
					Bucket:  "second",
					Content: "Второй...",
				},
			},
		},
		&RunForTestingTest{
			CaseName: "all filled, files contains not exists bucket",
			AvailableBuckets: []string{
				"first",
				"second",
				"third",
			},
			InitialFiles: []miniocontainer.File{
				{
					Name:    "Gagarin.pdf",
					Bucket:  "first",
					Content: "Поехали!",
				},
				{
					Name:    "Titov.pdf",
					Bucket:  "second",
					Content: "Второй...",
				},
				{
					Name:    "Tereshkova.pdf",
					Bucket:  "woman",
					Content: "Женщина на корабле к беде",
				},
			},
		},
	)
}
