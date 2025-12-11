package sync

import (
	"context"
	"fmt"
	"testing"
)

func TestGroupWithContext(t *testing.T) {
	res, err := GroupWithCancel(
		context.Background(),
		func(ctx context.Context) (int, error) {
			return 1, nil
		},
		func(ctx context.Context) (int, error) {
			return 2, nil
		},
		func(ctx context.Context) (int, error) {
			return 3, nil
		},
	)
	fmt.Println(res)
	fmt.Println(err)
}
