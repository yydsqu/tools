package utils

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

type Backoff struct {
	attempt uint64
	Factor  float64
	Jitter  bool
	Min     time.Duration
	Max     time.Duration
}

func (b *Backoff) Duration() time.Duration {
	d := b.ForAttempt(float64(atomic.AddUint64(&b.attempt, 1) - 1))
	return d
}

func (b *Backoff) Reset() {
	atomic.StoreUint64(&b.attempt, 0)
}

func (b *Backoff) Attempt() float64 {
	return float64(atomic.LoadUint64(&b.attempt))
}

const maxInt64 = float64(math.MaxInt64 - 512)

func (b *Backoff) ForAttempt(attempt float64) time.Duration {
	duration := b.Min
	if duration <= 0 {
		duration = 100 * time.Millisecond
	}
	m := b.Max
	if m <= 0 {
		m = 15 * time.Second
	}
	if duration >= m {
		return m
	}
	factor := b.Factor
	if factor <= 0 {
		factor = 2
	}
	minf := float64(duration)
	durf := minf * math.Pow(factor, attempt)
	if b.Jitter {
		durf = rand.Float64()*(durf-minf) + minf
	}
	if durf > maxInt64 {
		return m
	}
	dur := time.Duration(durf)
	if dur < duration {
		return duration
	}
	if dur > m {
		return m
	}
	return dur
}
