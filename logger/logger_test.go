package logger

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
)

func sss() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()
	fmt.Println("Done")
}

func TestTrace(t *testing.T) {

}
