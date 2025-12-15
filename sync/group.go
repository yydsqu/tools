package sync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

// GroupWithContext
// 获取到任意成功的结果但是不自动取消其他任务
func GroupWithContext[T any](ctx context.Context, fns ...func(ctx context.Context) (T, error)) ([]T, error) {
	if len(fns) == 0 {
		return nil, nil
	}
	wg := sync.WaitGroup{}
	ch := make(chan *Result[T], len(fns))

	for _, fn := range fns {
		wg.Add(1)
		go func(fn func(ctx context.Context) (T, error)) {
			defer wg.Done()
			defer func() {
				if recove := recover(); recove != nil {
					ch <- &Result[T]{Err: fmt.Errorf("panic: %v\n stack:%s", recove, debug.Stack())}
				}
			}()
			r, err := fn(ctx)
			ch <- &Result[T]{r, err}
		}(fn)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	results := make([]T, 0, len(fns))
	errs := make([]error, 0)

	for r := range ch {
		if r.Err == nil {
			results = append(results, r.Val)
		} else {
			errs = append(errs, r.Err)
		}
	}

	return results, errors.Join(errs...)
}

// GroupGenericWithContext
// 获取到任意成功的结果但是不自动取消其他任务
func GroupGenericWithContext[P any, R any](ctx context.Context, seeds []P, fn func(ctx context.Context, seed P) (R, error)) ([]R, error) {
	if len(seeds) == 0 {
		return nil, nil
	}

	wg := sync.WaitGroup{}
	ch := make(chan *Result[R], len(seeds))

	wg.Add(len(seeds))

	for _, seed := range seeds {
		go func(ctx context.Context, seed P) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					ch <- &Result[R]{
						Err: fmt.Errorf("panic seed=%v: %v\nstack:\n%s", seed, rec, debug.Stack()),
					}
				}
			}()
			r, err := fn(ctx, seed)
			ch <- &Result[R]{r, err}
		}(ctx, seed)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	results := make([]R, 0, len(seeds))
	errs := make([]error, 0)

	for r := range ch {
		if r.Err == nil {
			results = append(results, r.Val)
		} else {
			errs = append(errs, r.Err)
		}
	}

	return results, errors.Join(errs...)
}
