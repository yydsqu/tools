package utils

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func exampleHandle(ctx context.Context) {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Handle stopped gracefully.")
			return
		default:
			fmt.Printf("Handle running iteration %d\n", i)
			if i == 1 {
				panic("simulated panic")
			}
			time.Sleep(1 * time.Second)
		}
	}
}

type CondWorker struct {
	*sync.Mutex
	*sync.Cond
	handles []Handle
	ctx     context.Context
	cancel  context.CancelCauseFunc
	ready   bool
}

func (worker *CondWorker) Start() {
	worker.Lock()
	defer worker.Unlock()
	worker.ready = true
	worker.Broadcast()
}

func (worker *CondWorker) run() {
	for i, handle := range worker.handles {
		go func(handle Handle, i int) {
			worker.Lock()
			defer worker.Unlock()
			for !worker.ready {
				worker.Wait()
			}
			handle(worker.ctx)
		}(handle, i)
	}
}

func (worker *CondWorker) Stop() {

}

func NewCondWorker(handles ...Handle) *CondWorker {
	ctx, cancel := context.WithCancelCause(context.Background())
	mutex := &sync.Mutex{}
	cond := &CondWorker{
		ctx:     ctx,
		cancel:  cancel,
		handles: handles,
		Mutex:   mutex,
		Cond:    sync.NewCond(mutex),
	}
	cond.run()
	return cond
}

func TestName(t *testing.T) {
	worker := NewCondWorker(exampleHandle, exampleHandle, exampleHandle)
	time.Sleep(time.Second)
	worker.Start()
	time.Sleep(time.Second * 10)
}
