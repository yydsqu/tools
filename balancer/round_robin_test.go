package balancer

import (
	"testing"
)

func TestPolling(t *testing.T) {
	polling, err := NewRoundRobin("1", "2")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		polling.Next()
	}
}

func BenchmarkRoundRobin(b *testing.B) {
	polling, _ := NewRoundRobin("1", "2")
	for i := 0; i < b.N; i++ {
		polling.Next()
	}
}
