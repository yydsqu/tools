package sync

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestRacerGenericWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := RacerGenericWithContext(
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
	time.Sleep(1 * time.Minute)
}

func BenchmarkName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RacerGenericWithContext(
			context.Background(),
			[]int{1, 2, 3},
			func(ctx context.Context, p int) (string, error) {
				return strconv.Itoa(p), nil
			},
		)
	}
}
