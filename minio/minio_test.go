package miniocontainer_test

import (
	"context"
	"io"
	"testing"

	miniocontainer "github.com/amidgo/containers/minio"
	"github.com/amidgo/tester"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
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

	for _, bucket := range r.AvailableBuckets {
		exists, err := minioClient.BucketExists(ctx, bucket)
		require.NoError(t, err)
		require.True(t, exists)
	}

	for _, initialFile := range r.InitialFiles {
		requireInitialFileExists(t, ctx, minioClient, initialFile)
	}
}

func requireInitialFileExists(t *testing.T, ctx context.Context, minioClient *minio.Client, initialFile miniocontainer.File) {
	t.Helper()

	object, err := minioClient.GetObject(ctx, initialFile.Bucket, initialFile.Name, minio.GetObjectOptions{})
	require.NoError(t, err)

	objectData, err := io.ReadAll(object)
	require.NoError(t, err)

	require.Equal(t, initialFile.Content, string(objectData))
}

func Test_Minio(t *testing.T) {
	t.Parallel()

	tester.RunNamedTesters(t,
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
