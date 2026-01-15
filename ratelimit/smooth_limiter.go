package ratelimit

import (
	"context"
	"fmt"
	"time"
)

type SmoothLimiter struct {
	interval time.Duration
	burst    int
	t        *time.Ticker
	ch       chan struct{}
}

func (l *SmoothLimiter) Allow() bool {
	select {
	case <-l.ch:
		return true
	default:
		return false
	}
}

func (l *SmoothLimiter) Wait(ctx context.Context) error {
	select {
	case <-l.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *SmoothLimiter) Stop() {
	l.t.Stop()
}

func (l *SmoothLimiter) Reset() {
	l.t.Reset(l.interval)
}

// NewSmoothLimiter
// qps:每秒速率
// burst:最大突发
func NewSmoothLimiter(qps int, burst int) (*SmoothLimiter, error) {
	if qps <= 0 {
		return nil, fmt.Errorf("qps must > 0")
	}

	if burst <= 0 {
		return nil, fmt.Errorf("burst must > 0")
	}

	l := &SmoothLimiter{
		interval: time.Second / time.Duration(qps),
		t:        time.NewTicker(time.Second / time.Duration(qps)),
		ch:       make(chan struct{}, burst),
	}

	for i := 0; i < burst; i++ {
		l.ch <- struct{}{}
	}

	go func() {
		for range l.t.C {
			select {
			case l.ch <- struct{}{}:
			}
		}
	}()

	return l, nil
}
