package log

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TName1() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("TName1")
}

func TName2() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("TName2")
}

func TestNewAsyncFileWriter(t *testing.T) {

}
