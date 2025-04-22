package utils

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Backoff struct {
	min  time.Duration
	max  time.Duration
	curr time.Duration
	mu   sync.Mutex
	rng  *rand.Rand
}

func NewBackoff(min, max time.Duration) *Backoff {
	return &Backoff{
		min:  min,
		max:  max,
		curr: min,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *Backoff) Next() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	jitter := b.min + time.Duration(b.rng.Int63n(int64(b.curr-b.min)+1))
	next := time.Duration(math.Min(float64(b.curr*2), float64(b.max)))
	b.curr = next
	return jitter
}

func (b *Backoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.curr = b.min
}
