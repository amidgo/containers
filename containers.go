package containers

import (
	"os"
	"strconv"
	"testing"
)

func SkipDisabled(t *testing.T) {
	env := os.Getenv("CONTAINERS_DISABLE_TESTING")

	disabled, _ := strconv.ParseBool(env)

	if disabled {
		t.Skipf("test skipped because CONTAINERS_DISABLE_TESTING=%s", env)
	}
}
