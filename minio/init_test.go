package miniocontainer_test

import (
	"embed"
	"slices"
	"testing"

	miniocontainer "github.com/amidgo/containers/minio"
)

//go:embed testdata/files/*
var embedFiles embed.FS

func Test_FilesForTesting(t *testing.T) {
	files := miniocontainer.MustFiles(embedFiles)

	if len(files) != 2 {
		t.Fatalf("inavlid files count, %d, %+v", len(files), files)
	}

	first := files[0]

	if first.Name != "first.txt" {
		t.Fatalf("invalid file name, expected 'first.txt', actual %s", first.Name)
	}

	if !slices.Equal(first.Content, []byte("first\n")) {
		t.Fatalf("invalid file content, expected %q, actual %q", first.Content, "first\n")
	}

	second := files[1]

	if second.Name != "second.txt" {
		t.Fatalf("invalid file name, expected 'second.txt', actual %s", second.Name)
	}

	if !slices.Equal(second.Content, []byte("second\n")) {
		t.Fatalf("invalid file content, expected %q, actual %q", first.Content, "second\n")
	}
}
