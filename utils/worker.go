package utils

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

var (
	Canceled = errors.New("context canceled")
	Restart  = errors.New("context restart")
	Finish   = errors.New("context done")
)

type Handle func(ctx context.Context)

type SingleWorker struct {
	handle Handle
	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (sw *SingleWorker) Start() {
	sw.start()
}

func (sw *SingleWorker) Stop(err error) {
	sw.cancel(err)
}

func (sw *SingleWorker) Restart() {
	sw.cancel(Restart)
	<-sw.ctx.Done()
	sw.Start()
}

func (sw *SingleWorker) Status() bool {
	select {
	case <-sw.ctx.Done():
		return false
	default:
		return true
	}
}

func (sw *SingleWorker) Done() <-chan struct{} {
	return sw.ctx.Done()
}

func (sw *SingleWorker) Err() error {
	return context.Cause(sw.ctx)
}

func (sw *SingleWorker) start() {
	if sw.handle == nil {
		return
	}
	select {
	case <-sw.ctx.Done():
	default:
		return
	}
	sw.ctx, sw.cancel = context.WithCancelCause(context.Background())
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

func NewSingleWorker(handle Handle) *SingleWorker {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	sw := &SingleWorker{
		handle: handle,
		ctx:    ctx,
		cancel: cancel,
	}
	return sw
}
