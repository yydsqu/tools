package balancer

import (
	"errors"
	"math/rand"
	"time"
)

type Random[T any] struct {
	items []T
	rnd   *rand.Rand
}

func (p *Random[T]) Items() []T {
	return p.items
}

func (p *Random[T]) Next() T {
	return p.items[p.rnd.Int()%len(p.items)]
}

func NewRandom[T any](items ...T) (*Random[T], error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}

	return &Random[T]{
		items: items,
		rnd:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func MustRandom[T any](items ...T) *Random[T] {
	if len(items) == 0 {
		panic("empty items")
	}
	return &Random[T]{
		items: items,
	}
}
