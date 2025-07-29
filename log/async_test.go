package log

import (
	"fmt"
	"testing"
)

func TestNewAsyncFileWriter(t *testing.T) {
	writer := NewAsyncFileWriter("./bsc.log", 4000, 1, 1)
	writer.Start()
	fmt.Println(writer)
}
