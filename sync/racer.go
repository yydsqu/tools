package sync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

// RacerGenericWithContext
// 获取到任意成功的结果直接返回
func RacerGenericWithContext[P any, R any](parent context.Context, params []P, fn func(ctx context.Context, param P) (R, error)) (R, error) {
	var zero R
	if len(params) == 0 {
		return zero, nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	var wg sync.WaitGroup
	ch := make(chan *Result[R], len(params))

	for _, param := range params {
		wg.Add(1)
		go func(param P) {
			defer wg.Done()
			defer func() {
				if recove := recover(); recove != nil {
					select {
					case <-ctx.Done():
					case ch <- &Result[R]{Err: fmt.Errorf("panic: %v\n stack:%s", recove, debug.Stack())}:
					}
				}
			}()
			r, err := fn(ctx, param)
			select {
			case <-ctx.Done():
				return
			case ch <- &Result[R]{r, err}:
			}
		}(param)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	errs := make([]error, 0, len(params))

	for r := range ch {
		if r.Err == nil {
			return r.Val, nil
		}
		errs = append(errs, r.Err)
	}

	return zero, errors.Join(errs...)
}
