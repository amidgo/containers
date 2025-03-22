package miniocontainer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amidgo/containers"
	"github.com/minio/minio-go/v7"
)

func ReuseForTesting(
	t *testing.T,
	reusable *Reusable,
	buckets ...Bucket,
) *minio.Client {
	containers.SkipDisabled(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	minioClient, term, err := Reuse(ctx, reusable, buckets...)
	t.Cleanup(term)

	if err != nil {
		t.Fatal(err)

		return nil
	}

	return minioClient
}

func Reuse(
	ctx context.Context,
	reusable *Reusable,
	buckets ...Bucket,
) (minioClinet *minio.Client, term func(), err error) {
	return reusable.run(ctx, buckets...)
}

const defaultDuration = time.Second

type ReusableOption func(*Reusable)

func WithWaitDuration(waitDuration time.Duration) ReusableOption {
	return func(r *Reusable) {
		r.daemonWaitDuration = waitDuration
	}
}

type Reusable struct {
	ccf CreateContainerFunc

	runDaemonOnce      sync.Once
	daemon             *containers.ReusableDaemon
	stopDaemon         context.CancelFunc
	daemonWaitDuration time.Duration
}

func NewReusable(ccf CreateContainerFunc, opts ...ReusableOption) *Reusable {
	reusable := &Reusable{
		ccf:                ccf,
		daemonWaitDuration: defaultDuration,
	}

	for _, op := range opts {
		op(reusable)
	}

	return reusable
}

func (r *Reusable) Terminate(ctx context.Context) error {
	r.stopDaemon()

	select {
	case <-r.daemon.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (r *Reusable) run(ctx context.Context, buckets ...Bucket) (client *minio.Client, term func(), err error) {
	r.runDaemonOnce.Do(r.runDaemon)

	cnt, err := r.enter(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("enter to reuse container, %w", err)
	}

	client, term, err = r.reuse(ctx, cnt, buckets...)
	if err != nil {
		return nil, term, fmt.Errorf("reuse container, %w", err)
	}

	return client, term, nil
}

func (r *Reusable) runDaemon() {
	ccf := func(ctx context.Context) (any, error) {
		return r.ccf(ctx)
	}

	ctx, cancel := context.WithCancel(context.Background())

	r.daemon = containers.RunReusableDaemon(ctx,
		r.daemonWaitDuration,
		ccf,
	)
	r.stopDaemon = cancel
}

func (r *Reusable) enter(ctx context.Context) (Container, error) {
	cnt, err := r.daemon.Enter(ctx)
	if err != nil {
		return nil, err
	}

	return cnt.(Container), nil
}

func (r *Reusable) reuse(
	ctx context.Context,
	cnt Container,
	buckets ...Bucket,
) (minioClient *minio.Client, term func(), err error) {
	term = r.daemon.Exit

	minioClient, err = cnt.Connect(ctx)
	if err != nil {
		return nil, term, fmt.Errorf("connect to container, %w", err)
	}

	err = insertBuckets(ctx, minioClient, buckets...)
	if err != nil {
		return nil, term, err
	}

	return minioClient, term, nil
}
