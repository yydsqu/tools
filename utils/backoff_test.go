package utils

import (
	"fmt"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	backoff := NewBackoff(time.Second, time.Minute)
	for i := 0; i < 10; i++ {
		fmt.Println(backoff.Next())
	}
}
