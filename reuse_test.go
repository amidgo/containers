package containers_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/amidgo/containers"
)

func Test_ReuseDaemon(t *testing.T) {
	t.Parallel()

	waitDuration := time.Second

	called := false
	cnt := "container"
	errDoubleCffCall := errors.New("unexpected, second call to ccf")

	ccf := containers.CreateContainerFunc(func(ctx context.Context) (any, error) {
		if called {
			return nil, errDoubleCffCall
		}

		called = true

		return cnt, nil
	})

	daemon := containers.NewReuseDaemon(waitDuration, ccf)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go daemon.Run(ctx)

	notifyCtx, cancel := context.WithCancel(ctx)

	t.Run("simple enter and exit", simpleEnterAndExit(daemon, cancel, cnt, waitDuration))
	t.Run("await notify, enter and exit", awaitNotifyEnterAndExit(daemon, notifyCtx, cnt))
}

func simpleEnterAndExit(
	daemon *containers.ReuseDaemon,
	notify func(),
	expectedCnt any,
	waitDuration time.Duration,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		cnt, err := daemon.Enter(ctx)
		if err != nil {
			t.Fatalf("enter to daemon, expected no error, actual %s", err)
		}

		if !reflect.DeepEqual(expectedCnt, cnt) {
			t.Fatalf("enter to daemon, expected %+v, actual %+v", expectedCnt, cnt)
		}

		go daemon.Exit()

		<-time.After(waitDuration / 2)

		notify()
	}
}

func awaitNotifyEnterAndExit(
	daemon *containers.ReuseDaemon,
	notifyCtx context.Context,
	expectedCnt any,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		<-notifyCtx.Done()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		cnt, err := daemon.Enter(ctx)
		if err != nil {
			t.Fatalf("enter to daemon, expected no error, actual %s", err)
		}

		if !reflect.DeepEqual(expectedCnt, cnt) {
			t.Fatalf("enter to daemon, expected %+v, actual %+v", expectedCnt, cnt)
		}

		daemon.Exit()
	}
}
