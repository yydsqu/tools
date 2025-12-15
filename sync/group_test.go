package sync

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGroupGenericWithContext(t *testing.T) {
	res, err := GroupGenericWithContext(
		context.Background(),
		[]int{1, 2, 3},
		func(ctx context.Context, seed int) (int, error) {
			time.Sleep(time.Second)
			return seed, nil
		},
	)
	fmt.Println(res)
	fmt.Println(err)
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
