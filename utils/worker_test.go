package utils

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func exampleHandle(ctx context.Context) {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Handle stopped gracefully.")
			return
		default:
			fmt.Printf("Handle running iteration %d\n", i)
			if i == 1 {
				panic("simulated panic")
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func TestName(t *testing.T) {
	sw := NewSingleWorker(exampleHandle)
	sw.Start()
	fmt.Println("等待结束")
	if err := sw.Wait(); err != nil {
		t.Fatal(err)
	}
}
