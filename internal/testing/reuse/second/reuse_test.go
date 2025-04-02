package first_test

import (
	"testing"

	"github.com/amidgo/containers/internal/testing/reuse"
)

func Test_First(t *testing.T) {
	t.Parallel()

	t.Run("zero user exit", reuse.ReuseDaemon_Zero_User_Exit)
}

func Test_Second(t *testing.T) {
	t.Parallel()

	t.Run("zero user exit", reuse.ReuseDaemon_Zero_User_Exit)
}

func Test_Third(t *testing.T) {
	t.Parallel()

	t.Run("zero user exit", reuse.ReuseDaemon_Zero_User_Exit)
}
