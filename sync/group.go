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
		wg.Go(func(fn func(ctx context.Context) (T, error)) func() {
			return func() {
				defer func() {
					if recove := recover(); recove != nil {
						ch <- &Result[T]{Err: fmt.Errorf("panic:%v\nstack:%s", recove, debug.Stack())}
					}
				}()
				r, err := fn(ctx)
				ch <- &Result[T]{r, err}
			}
		}(fn))
	}

	wg.Wait()
	close(ch)

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
// 获取全部的结果和错误
func GroupGenericWithContext[P any, R any](ctx context.Context, seeds []P, fn func(ctx context.Context, seed P) (R, error)) ([]R, error) {
	if len(seeds) == 0 {
		return nil, nil
	}

	wg := sync.WaitGroup{}
	ch := make(chan *Result[R], len(seeds))

	for _, seed := range seeds {
		wg.Go(func(seed P) func() {
			return func() {
				defer func() {
					if recove := recover(); recove != nil {
						ch <- &Result[R]{Err: fmt.Errorf("seed:%v\npanic: %v\nstack:%s", seed, recove, debug.Stack())}
					}
				}()
				r, err := fn(ctx, seed)
				ch <- &Result[R]{r, err}
			}
		}(seed))
	}

	wg.Wait()
	close(ch)

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
