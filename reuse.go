package containers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

type reuseCommand uint8

const (
	reuseCommandEnter reuseCommand = iota
	reuseCommandExit
)

type reuseContainerRequest struct {
	reuseCmd reuseCommand
	ctx      context.Context
}

type reuseContainerResponse struct {
	cnt any
	err error
}

type CreateContainerFunc func(ctx context.Context) (any, error)

type ReuseDaemon struct {
	activeUsers  int
	cnt          any
	waitDuration time.Duration
	stopped      atomic.Bool
	mainCtx      context.Context
	termCtx      context.Context

	reqCh  chan reuseContainerRequest
	respCh chan reuseContainerResponse

	ccf CreateContainerFunc
}

func RunReuseDaemon(
	ctx context.Context,
	waitDuration time.Duration,
	ccf CreateContainerFunc,
) *ReuseDaemon {
	termCtx, cancel := context.WithCancel(context.Background())

	daemon := &ReuseDaemon{
		waitDuration: waitDuration,
		reqCh:        make(chan reuseContainerRequest),
		respCh:       make(chan reuseContainerResponse),
		ccf:          ccf,
		mainCtx:      ctx,
		termCtx:      termCtx,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				daemon.clearContainer(termCtx)
				cancel()

				return
			case req := <-daemon.reqCh:
				daemon.handleReuseCommand(req.ctx, req.reuseCmd)
			}
		}
	}()

	return daemon
}

func (d *ReuseDaemon) Enter(ctx context.Context) (any, error) {
	select {
	case <-d.mainCtx.Done():
		return nil, fmt.Errorf("root ctx is done, %w", context.Cause(d.mainCtx))
	case d.reqCh <- reuseContainerRequest{
		ctx:      ctx,
		reuseCmd: reuseCommandEnter,
	}:
		resp := <-d.respCh

		return resp.cnt, resp.err
	}
}

func (d *ReuseDaemon) Exit() {
	select {
	case <-d.mainCtx.Done():
		<-d.termCtx.Done()
	case d.reqCh <- reuseContainerRequest{
		ctx:      context.Background(),
		reuseCmd: reuseCommandExit,
	}:
		<-d.respCh
	}
}

func (d *ReuseDaemon) handleReuseCommand(ctx context.Context, reuseCmd reuseCommand) {
	switch reuseCmd {
	case reuseCommandEnter:
		d.activeUsers++
	case reuseCommandExit:
		d.activeUsers--
	default:
		panic("invalid reuse command received: " + strconv.FormatUint(uint64(reuseCmd), 10))
	}

	switch {
	case d.activeUsers > 0:
		d.handlePositiveActiveUsers(ctx)
	case d.activeUsers == 0:
		d.handleZeroActiveUsers(ctx)
	case d.activeUsers <= 0:
		panic("reuse container term func called twice, negative amount of active users")
	}
}

func (d *ReuseDaemon) handlePositiveActiveUsers(ctx context.Context) {
	if d.cnt == nil {
		cnt, err := d.ccf(ctx)
		if err != nil {
			d.respCh <- reuseContainerResponse{
				err: fmt.Errorf("create new container, %w", err),
			}

			return
		}

		d.cnt = cnt
	}

	d.respCh <- reuseContainerResponse{
		cnt: d.cnt,
	}
}

func (d *ReuseDaemon) handleZeroActiveUsers(ctx context.Context) {
	select {
	case <-time.After(d.waitDuration):
		d.clearContainer(ctx)
		d.respCh <- reuseContainerResponse{}
	case req := <-d.reqCh:
		switch req.reuseCmd {
		case reuseCommandEnter:
			d.activeUsers++
			d.respCh <- reuseContainerResponse{
				cnt: d.cnt,
			}
			d.respCh <- reuseContainerResponse{
				cnt: d.cnt,
			}
		case reuseCommandExit:
			panic("unexpected exit command in handleZeroActiveUsers")
		default:
			panic("invalid reuse command received: " + strconv.FormatUint(uint64(req.reuseCmd), 10))
		}
	}
}

func (d *ReuseDaemon) clearContainer(ctx context.Context) {
	if d.cnt == nil {
		return
	}

	type Terminater interface {
		Terminate(ctx context.Context) error
	}

	trm, ok := d.cnt.(Terminater)
	if ok {
		err := trm.Terminate(ctx)
		if err != nil {
			log.Printf("failed terminate container, %s", err)
		}
	}

	d.cnt = nil
}
