package balancer

import (
	"fmt"
	"testing"
)

func TestRandom(t *testing.T) {
	polling, _ := NewRandom("1", "2")
	for i := 0; i < 100; i++ {
		fmt.Println(polling.Next())
	}
}

func BenchmarkRandom(b *testing.B) {
	polling := MustRandom("1", "2")
	for i := 0; i < b.N; i++ {
		polling.Next()
	}
}
