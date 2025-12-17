package sync

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGroupGenericWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := GroupGenericWithContext(
		ctx,
		[]int{0, 1, 2, 3, 4, 5, 6, 7, 8},
		func(ctx context.Context, seed int) (int, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(time.Duration(seed) * time.Second):
				return 10 % seed, nil
			}
		},
	)
	fmt.Println(res)
	fmt.Println(err)
	fmt.Println("=================")
}

func BenchmarkGroupGenericWithContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GroupGenericWithContext(
			b.Context(),
			[]int{1, 2, 3},
			func(ctx context.Context, seed int) (int, error) {
				return seed, nil
			},
		)
	}
}
