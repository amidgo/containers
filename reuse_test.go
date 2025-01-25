package containers_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amidgo/containers"
)

type mockTerminater struct {
	t          *testing.T
	terminated atomic.Bool
}

func newMockTerminater(t *testing.T) *mockTerminater {
	term := &mockTerminater{
		t: t,
	}

	t.Cleanup(term.assert)

	return term
}

func (m *mockTerminater) Terminate(context.Context) error {
	swapped := m.terminated.CompareAndSwap(false, true)

	if !swapped {
		m.t.Fatal("mockTerminater.Terminate called twice")
	}

	return nil
}

func (m *mockTerminater) assert() {
	terminated := m.terminated.Load()

	if !terminated {
		m.t.Fatal("assert mockTerminater is terminated failed")
	}
}

func Test_ReuseDaemon_Zero_User_Exit(t *testing.T) {
	t.Parallel()

	waitDuration := time.Second

	called := false
	cnt := newMockTerminater(t)
	errDoubleCffCall := errors.New("unexpected, second call to ccf")

	ccf := containers.CreateContainerFunc(func(ctx context.Context) (any, error) {
		if called {
			return nil, errDoubleCffCall
		}

		called = true

		return cnt, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	daemon := containers.RunReuseDaemon(ctx, waitDuration, ccf)

	notifyCtx, cancel := context.WithCancel(ctx)

	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()

		simpleEnterAndExit(t, daemon, cancel, cnt, waitDuration)
	}()

	go func() {
		defer wg.Done()

		awaitNotifyEnterAndExit(t, daemon, notifyCtx, cnt)
	}()

	wg.Wait()
}

func simpleEnterAndExit(
	t *testing.T,
	daemon *containers.ReuseDaemon,
	notify func(),
	expectedCnt any,
	waitDuration time.Duration,
) {
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

func awaitNotifyEnterAndExit(
	t *testing.T,
	daemon *containers.ReuseDaemon,
	notifyCtx context.Context,
	expectedCnt any,
) {
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

func Test_ReuseDaemon_Cancel_Root_Context(t *testing.T) {
	t.Parallel()

	t.Run("canceled ctx", canceledCtx)
}

func canceledCtx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rootCtx, rootCancel := context.WithCancel(context.Background())

	cnt := newMockTerminater(t)

	called := false

	ccf := containers.CreateContainerFunc(func(ctx context.Context) (any, error) {
		if called {
			t.Fatal("ccf called twice")
		}

		called = true

		return cnt, nil
	})

	waitDuration := time.Duration(-1)

	daemon := containers.RunReuseDaemon(
		rootCtx,
		waitDuration,
		ccf,
	)

	enterCnt, err := daemon.Enter(ctx)
	if err != nil {
		t.Fatalf("expected no error, actual %+v", err)
	}

	if !reflect.DeepEqual(enterCnt, cnt) {
		t.Fatalf("wrong enterCnt, expected %+v, actual %+v", cnt, enterCnt)
	}

	rootCancel()

	enterCnt, err = daemon.Enter(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"unexpected error of entering in canceled daemon, expected context.Canceled, actual %+v",
			err,
		)
	}

	if enterCnt != nil {
		t.Fatalf("unexpected cnt, expected nil, actual %+v", cnt)
	}

	daemon.Exit()

	enterCnt, err = daemon.Enter(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"unexpected error of entering in canceled daemon, expected context.Canceled, actual %+v",
			err,
		)
	}
}
