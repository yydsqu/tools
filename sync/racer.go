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
func RacerGenericWithContext[P any, R any](parent context.Context, seeds []P, fn func(ctx context.Context, seed P) (R, error)) (R, error) {
	var zero R
	if len(seeds) == 0 {
		return zero, nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	var wg sync.WaitGroup
	ch := make(chan *Result[R], len(seeds))

	for _, seed := range seeds {
		wg.Go(func(seed P) func() {
			return func() {
				defer func() {
					if recove := recover(); recove != nil {
						select {
						case ch <- &Result[R]{Err: fmt.Errorf("seed:%v\npanic: %v\n stack:%s", seed, recove, debug.Stack())}:
						case <-ctx.Done():
						}
					}
				}()
				r, err := fn(ctx, seed)
				select {
				case <-ctx.Done():
					return
				case ch <- &Result[R]{r, err}:
				}

			}
		}(seed))
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	errs := make([]error, 0, len(seeds))

	for r := range ch {
		if r.Err == nil {
			return r.Val, nil
		}
		errs = append(errs, r.Err)
	}

	return zero, errors.Join(errs...)
}
