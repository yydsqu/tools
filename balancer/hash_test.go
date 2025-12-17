package balancer

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	polling, _ := NewHash("1", "2")
	for i := 0; i < 100; i++ {
		fmt.Println(polling.Next(strconv.Itoa(i)))
	}
}

func BenchmarkHash(b *testing.B) {
	polling, _ := NewHash("1", "2")
	for i := 0; i < b.N; i++ {
		polling.Next(strconv.Itoa(i))
	}
}
