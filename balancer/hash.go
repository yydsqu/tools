package balancer

import (
	"errors"
	"hash/crc32"
)

type Hash[T any] struct {
	items []T
}

func (p *Hash[T]) Items() []T {
	return p.items
}

func (p *Hash[T]) Next(key string) T {
	return p.items[crc32.ChecksumIEEE([]byte(key))%uint32(len(p.items))]
}

func NewHash[T any](items ...T) (*Hash[T], error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}
	return &Hash[T]{
		items: items,
	}, nil
}
