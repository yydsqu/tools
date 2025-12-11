package sync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

// Group
// 等待获取全部异步结果
func Group[T any](tasks ...func() (T, error)) ([]T, error) {
	var (
		ch      = make(chan *Result[T], len(tasks))
		results []T
		errs    []error
	)

	for _, task := range tasks {
		go func(fn func() (T, error)) {
			defer func() {
				if recove := recover(); recove != nil {
					select {
					case ch <- &Result[T]{Err: fmt.Errorf("panic: %v\n stack:%s", recove, debug.Stack())}:
					default:
					}
				}
			}()
			r, err := fn()
			select {
			case ch <- &Result[T]{r, err}:
			default:

			}
		}(task)
	}

	for i := 0; i < len(tasks); i++ {
		r := <-ch
		if r.Err == nil {
			results = append(results, r.Val)
		} else {
			errs = append(errs, r.Err)
		}
	}

	return results, errors.Join(errs...)
}

// GroupWithContext
// 获取到任意成功的结果但是不自动取消其他任务
func GroupWithContext[T any](ctx context.Context, tasks ...func(ctx context.Context) (T, error)) ([]T, error) {
	var (
		ch      = make(chan *Result[T], len(tasks))
		results []T
		errs    []error
	)

	for _, task := range tasks {
		go func(fn func(ctx context.Context) (T, error)) {
			defer func() {
				if recove := recover(); recove != nil {
					ch <- &Result[T]{Err: fmt.Errorf("panic: %v\n stack:%s", recove, debug.Stack())}
				}
			}()
			r, err := fn(ctx)
			ch <- &Result[T]{r, err}
		}(task)
	}

	for i := 0; i < len(tasks); i++ {
		r := <-ch
		if r.Err == nil {
			results = append(results, r.Val)
		} else {
			errs = append(errs, r.Err)
		}
	}

	return results, errors.Join(errs...)
}

// GroupWithCancel
// 自动取消任务的
func GroupWithCancel[T any](parentCtx context.Context, tasks ...func(ctx context.Context) (T, error)) ([]T, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	return GroupWithContext(ctx, tasks...)
}

// GroupWithTimeout
// 控制总超时时间
func GroupWithTimeout[T any](timeout time.Duration, tasks ...func(ctx context.Context) (T, error)) ([]T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return GroupWithContext(ctx, tasks...)
}
