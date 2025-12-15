package retry

import (
	"context"
	"math/rand"
	"time"
)

var (
	DefaultJitter = 0.5
)

func Do(ctx context.Context, attempts int, delay time.Duration, fn func() error) (err error) {
	var timer *time.Timer

	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err = fn(); err == nil {
			return nil
		}

		// 最后一次失败：直接返回，不再等待
		if attempt == attempts || delay <= 0 {
			continue
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			stopTimer()
			timer.Reset(WithJitter(delay, DefaultJitter))
		}

		select {
		case <-ctx.Done():
			stopTimer()
			return ctx.Err()
		case <-timer.C:

		}
	}

	return err
}

func DoWithResult[R any](ctx context.Context, attempts int, delay time.Duration, fn func() (r R, err error)) (r R, err error) {
	var timer *time.Timer

	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}

		if r, err = fn(); err == nil {
			return r, nil
		}

		// 最后一次失败：直接返回，不再等待
		if attempt == attempts || delay <= 0 {
			continue
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			stopTimer()
			timer.Reset(WithJitter(delay, DefaultJitter))
		}

		select {
		case <-ctx.Done():
			stopTimer()
			return r, ctx.Err()
		case <-timer.C:

		}
	}

	return
}

func DoWithJitter[R any](ctx context.Context, attempts int, jitter float64, delay time.Duration, fn func() (r R, err error)) (r R, err error) {
	var timer *time.Timer

	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}

		if r, err = fn(); err == nil {
			return r, nil
		}

		// 最后一次失败：直接返回，不再等待
		if attempt == attempts || delay <= 0 {
			continue
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			stopTimer()
			timer.Reset(WithJitter(delay, jitter))
		}

		select {
		case <-ctx.Done():
			stopTimer()
			return r, ctx.Err()
		case <-timer.C:

		}
	}

	return
}

func WithJitter(d time.Duration, jitter float64) time.Duration {
	if d <= 0 || jitter <= 0 {
		return d
	}
	if jitter > 1 {
		jitter = 1
	}
	factor := (1 - jitter) + rand.Float64()*(2*jitter)
	return time.Duration(float64(d) * factor)
}
