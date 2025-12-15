package sync

import (
	"context"
	"strconv"
	"testing"
)

func TestRacerGenericWithContext(t *testing.T) {

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
