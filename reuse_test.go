package containers_test

import (
	"context"
	"errors"
	"math/rand/v2"
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

	daemon := containers.RunReusableDaemon(ctx, waitDuration, ccf)

	notifyCtx, notify := context.WithCancel(ctx)

	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		defer wg.Done()

		simpleEnterAndExit(t, daemon, notify, cnt, waitDuration)
	}()

	go func() {
		defer wg.Done()

		awaitNotifyEnterAndExit(t, daemon, notifyCtx, cnt)
	}()

	wg.Wait()
}

func Test_ManyConcurrentEnterAndExit(t *testing.T) {
	t.Parallel()

	waitDuration := time.Second

	called := false
	errDoubleCffCall := errors.New("unexpected, second call to ccf")

	ccf := containers.CreateContainerFunc(func(ctx context.Context) (any, error) {
		<-time.After(time.Second * 2)

		if called {
			return false, errDoubleCffCall
		}

		called = true

		return true, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	daemon := containers.RunReusableDaemon(ctx, waitDuration, ccf)

	count := 1000
	if testing.Short() {
		count = 10
	}

	runConcurrentEnters(t, ctx, daemon, count)
}

func runConcurrentEnters(t *testing.T, ctx context.Context, daemon *containers.ReusableDaemon, count int) {
	sendCh := make(chan struct{})
	defer close(sendCh)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	for range count {
		go func() {
			_, err := daemon.Enter(ctx)
			if err != nil {
				cancel(err)
			}

			sleepDuration := time.Duration(rand.IntN(1000)) * time.Millisecond
			<-time.After(sleepDuration)

			sendCh <- struct{}{}
		}()
	}

	entered := 0

	for {
		select {
		case <-ctx.Done():
			t.Fatal(context.Cause(ctx))

			return
		case <-sendCh:
			entered++

			daemon.Exit()

			if entered == count {
				return
			}
		}
	}
}

func simpleEnterAndExit(
	t *testing.T,
	daemon *containers.ReusableDaemon,
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
	daemon *containers.ReusableDaemon,
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
	t.Run("in time canceled ctx", inTimeCanceledCtx)
}

func canceledCtx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rootCtx, rootCancel := context.WithCancel(ctx)

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

	daemon := containers.RunReusableDaemon(
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

func inTimeCanceledCtx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	waitDuration := time.Millisecond

	ccf := containers.CreateContainerFunc(func(ctx context.Context) (any, error) {
		return newMockTerminater(t), nil
	})

	runInTimeCanceledReuseDaemon(
		t,
		ctx,
		waitDuration,
		ccf,
	)
}

func runInTimeCanceledReuseDaemon(
	t *testing.T,
	ctx context.Context,
	waitDuration time.Duration,
	ccf containers.CreateContainerFunc,
) {
	rootCtx, rootCancel := context.WithCancel(ctx)

	daemon := containers.RunReusableDaemon(
		rootCtx,
		waitDuration,
		ccf,
	)

	timeCtx, timeCancel := context.WithTimeout(ctx, time.Millisecond*10)
	t.Cleanup(timeCancel)

	go func() {
		<-timeCtx.Done()
		rootCancel()
	}()

	<-timeCtx.Done()

	_, err := daemon.Enter(ctx)
	switch err {
	case nil:
		daemon.Exit()

		runInTimeCanceledReuseDaemon(
			t,
			ctx,
			waitDuration,
			ccf,
		)
	default:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error, expected context.Canceled, actual %+v", err)
		}

		runInTimeCanceledReuseDaemonWhileCanceled(
			t,
			ctx,
			waitDuration,
			ccf,
		)
	}
}

func runInTimeCanceledReuseDaemonWhileCanceled(
	t *testing.T,
	ctx context.Context,
	waitDuration time.Duration,
	ccf containers.CreateContainerFunc,
) {
	rootCtx, rootCancel := context.WithCancel(ctx)

	daemon := containers.RunReusableDaemon(
		rootCtx,
		waitDuration,
		ccf,
	)

	timeCtx, timeCancel := context.WithTimeout(ctx, time.Millisecond*10)
	t.Cleanup(timeCancel)

	go func() {
		<-timeCtx.Done()
		rootCancel()
	}()

	<-timeCtx.Done()

	_, err := daemon.Enter(ctx)
	switch err {
	case nil:
		daemon.Exit()
	default:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error, expected context.Canceled, actual %+v", err)
		}

		runInTimeCanceledReuseDaemonWhileCanceled(
			t,
			ctx,
			waitDuration,
			ccf,
		)
	}
}
