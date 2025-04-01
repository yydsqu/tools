package utils

import (
	"context"
	"fmt"
	"github.com/yydsqu/tools/log"
	"runtime"
	"sync"
)

type Handle func(ctx context.Context)

type SingleWorker struct {
	handle  Handle
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelCauseFunc
	mu      sync.Mutex
	err     error
	running bool
}

func (sw *SingleWorker) SetHandle(handle Handle) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.handle = handle
}

func (sw *SingleWorker) StartHandle(handle Handle) {
	sw.SetHandle(handle)
	sw.Start()
}

func (sw *SingleWorker) RestartHandle(handle Handle) {
	sw.Stop()
	sw.StartHandle(handle)
}

func (sw *SingleWorker) Start() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.start()
}

func (sw *SingleWorker) Restart() {
	sw.Stop()
	sw.Start()
}

func (sw *SingleWorker) Stop() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if sw.running {
		sw.cancel(context.Canceled)
		sw.running = false
	}
}

func (sw *SingleWorker) Error(err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if sw.running {
		sw.cancel(err)
		sw.running = false
	}
}

func (sw *SingleWorker) Status() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.running
}

func (sw *SingleWorker) Wait() error {
	sw.wg.Wait()
	return sw.err
}

func (sw *SingleWorker) start() {
	if sw.running {
		log.Trace("worker is already running")
		return
	}
	if sw.handle == nil {
		log.Trace("worker handle is nil")
		return
	}
	// 等待其他进程退出
	sw.wg.Wait()
	sw.running = true
	sw.wg.Add(1)
	sw.ctx, sw.cancel = context.WithCancelCause(context.Background())
	sw.err = nil

	go func() {
		defer func() {
			if err := Recover(); err != nil {
				fmt.Println(err)
				log.Error("worker recover panic", "err", err)
				sw.err = err
			}
			sw.wg.Done()
			sw.Stop()
		}()
		sw.handle(sw.ctx)
	}()
}

func NewSingleWorker(handle Handle) *SingleWorker {
	ctx, cancel := context.WithCancelCause(context.Background())
	sw := &SingleWorker{
		handle: handle,
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}
	return sw
}

func Recover() error {
	if r := recover(); r != nil {
		buf := make([]byte, 1024)
		n := runtime.Stack(buf, false)
		return fmt.Errorf("panic: %v\nstack trace:\n%s", r, buf[:n])
	}
	return nil
}

func PrintStack() {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	fmt.Printf("stack trace:\n%s\n", buf[:n])
}
