package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestWithJitter(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(int64(WithJitter(10, 0.2)))
	}
}

func TestDo(t *testing.T) {
	result, err := DoWithJitter(context.Background(), 1, 0.01, time.Second, func() (int, error) {
		intn := rand.Intn(10)
		t.Log(intn)
		if intn%2 == 0 {
			return 0, errors.New("err")
		}
		return intn, nil
	})

	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(result)
}
