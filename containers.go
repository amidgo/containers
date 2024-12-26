package containers

import (
	"os"
	"strconv"
	"testing"
)

func Disabled() bool {
	env := os.Getenv("CONTAINERS_DISABLE_TESTING")

	disabled, _ := strconv.ParseBool(env)

	return disabled
}

func SkipDisabled(t *testing.T) {
	if Disabled() {
		t.Skipf("test skipped because CONTAINERS_DISABLE_TESTING is SET to TRUE")
	}
}
