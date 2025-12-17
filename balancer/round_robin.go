package balancer

import (
	"errors"
	"sync"
)

type RoundRobin[T any] struct {
	items []T
	index int
	mutex sync.RWMutex
}

func (p *RoundRobin[T]) Items() []T {
	return p.items
}

func (p *RoundRobin[T]) Next() T {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	v := p.items[p.index]
	p.index = (p.index + 1) % len(p.items)
	return v
}

func NewRoundRobin[T any](items ...T) (*RoundRobin[T], error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}
	return &RoundRobin[T]{
		items: items,
	}, nil
}
