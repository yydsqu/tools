package utils

import (
	"fmt"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	b := &Backoff{
		attempt: 3,
		Min:     100 * time.Millisecond,
		Max:     15 * time.Second,
		Factor:  1.75,
		Jitter:  true,
	}
	for i := 0; i < 100; i++ {
		fmt.Println(b.Duration())
	}
}
