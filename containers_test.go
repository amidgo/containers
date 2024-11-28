package containers_test

import (
	"testing"

	"github.com/amidgo/containers"
)

func Test_Skipped(t *testing.T) {
	t.Setenv("CONTAINERS_DISABLE_TESTING", "true")

	containers.SkipDisabled(t)

	t.Fatal("expected test is skipped")
}
