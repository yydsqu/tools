package worker

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
)

var (
	Canceled = errors.New("context canceled")
	Restart  = errors.New("context restart")
	Finish   = errors.New("context done")
)

type Handle func(ctx context.Context)

type Worker struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	handle Handle
	mutex  sync.Mutex
}

func (sw *Worker) Start() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.ctx != nil {
		select {
		case <-sw.ctx.Done():
		default:
			return
		}
	}
	sw.ctx, sw.cancel = context.WithCancelCause(context.Background())
	sw.start()
}

func (sw *Worker) Stop(err error) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.cancel != nil {
		sw.cancel(err)
	}
}

func (sw *Worker) Restart(wait bool) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.cancel != nil {
		sw.cancel(Restart)
	}
	if sw.ctx != nil && wait {
		<-sw.ctx.Done()
	}
	sw.ctx, sw.cancel = context.WithCancelCause(context.Background())
	sw.start()
}

func (sw *Worker) Status() bool {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.ctx != nil {
		select {
		case <-sw.ctx.Done():
			return false
		default:
			return true
		}
	}
	return false
}

func (sw *Worker) Done() <-chan struct{} {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.ctx != nil {
		return sw.ctx.Done()
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (sw *Worker) Err() error {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	if sw.ctx == nil {
		return nil
	}
	return context.Cause(sw.ctx)
}

func (sw *Worker) Wait() error {
	sw.mutex.Lock()
	ctx := sw.ctx
	sw.mutex.Unlock()
	if ctx == nil {
		return nil
	}
	<-ctx.Done()
	return sw.Err()
}

func (sw *Worker) start() {
	if sw.handle == nil || sw.ctx == nil || sw.cancel == nil {
		return
	}
	go func(ctx context.Context, cancel context.CancelCauseFunc) {
		defer func() {
			var err error
			if r := recover(); r != nil {
				buf := make([]byte, 1024)
				n := runtime.Stack(buf, false)
				err = fmt.Errorf("panic: %v\nstack trace:\n%s", r, buf[:n])
			}
			cancel(err)
		}()
		sw.handle(ctx)
	}(sw.ctx, sw.cancel)
}

func NewWorker(handle Handle) *Worker {
	return &Worker{
		handle: handle,
	}
}
