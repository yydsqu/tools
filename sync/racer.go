package sync

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
)

// Racer
// 最简单获取最快的结果
func Racer[T any](tasks ...func() (T, error)) (T, error) {
	ch := make(chan *Result[T], len(tasks))

	for _, task := range tasks {
		go func(fn func() (T, error)) {
			r, err := fn()
			ch <- &Result[T]{r, err}
		}(task)
	}

	var errs []error
	var zero T

	for i := 0; i < len(tasks); i++ {
		r := <-ch
		if r.Err == nil {
			return r.Val, nil
		}
		errs = append(errs, r.Err)
	}

	return zero, errors.Join(errs...)
}

// RacerWithContext
// 获取到任意成功的结果但是不自动取消其他任务
func RacerWithContext[R any](ctx context.Context, tasks ...func(ctx context.Context) (R, error)) (R, error) {
	var (
		ch   = make(chan *Result[R])
		zero R
		errs []error
	)

	for _, task := range tasks {
		go func(fn func(ctx context.Context) (R, error)) {
			// 控制错误信息
			defer func() {
				if recove := recover(); recove != nil {
					select {
					case <-ctx.Done():
					case ch <- &Result[R]{Err: fmt.Errorf("panic: %v\n stack:%s", recove, debug.Stack())}:
					}
				}
			}()

			r, err := fn(ctx)
			select {
			case <-ctx.Done():
			case ch <- &Result[R]{r, err}:
			}
		}(task)
	}

	for i := 0; i < len(tasks); i++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case r := <-ch:
			if r.Err == nil {
				return r.Val, nil
			}
			errs = append(errs, r.Err)
		}
	}

	return zero, errors.Join(errs...)
}

// RacerWithCancel
// 获取到任意成功的结果自动取消其他任务
func RacerWithCancel[T any](parent context.Context, tasks ...func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	return RacerWithContext(ctx, tasks...)
}

// RacerWithTimeout
// 控制总超时时间
func RacerWithTimeout[T any](parent context.Context, timeout time.Duration, tasks ...func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return RacerWithContext(ctx, tasks...)
}
