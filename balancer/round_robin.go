package balancer

import (
	"errors"
	"sync/atomic"
)

type RoundRobin[T any] struct {
	items []T
	index atomic.Int64
}

func (p *RoundRobin[T]) Items() []T {
	return p.items
}

func (p *RoundRobin[T]) Next() T {
	return p.items[int(p.index.Add(1))%len(p.items)]
}

func NewRoundRobin[T any](items ...T) (*RoundRobin[T], error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}
	return &RoundRobin[T]{
		items: items,
	}, nil
}

func MustRoundRobin[T any](items ...T) *RoundRobin[T] {
	if len(items) == 0 {
		panic("empty items")
	}
	return &RoundRobin[T]{
		items: items,
		index: atomic.Int64{},
	}
}
