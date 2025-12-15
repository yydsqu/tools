package worker

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func handle(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	fmt.Println("开始运行", time.Now())
	i := 1
	for {
		select {
		case <-ctx.Done():
			fmt.Println("完成", i)
			return
		case <-ticker.C:
			i++
			fmt.Println("==========", i)
		}
	}
}

func TestSingleWorker(t *testing.T) {
	worker := NewWorker(handle)
	fmt.Println("0、===============", time.Now())
	worker.Start()
	fmt.Println("1、===============", time.Now())
	worker.Start()
	time.Sleep(time.Second * 3)
	fmt.Println("3、===============", time.Now())
	worker.Restart(false)
	time.Sleep(time.Hour)
}
