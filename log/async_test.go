package log

import (
	"fmt"
	"testing"
)

func TestNewAsyncFileWriter(t *testing.T) {
	config := Config{
		Output:     "./logs/log.log",
		UseColor:   false,
		MaxBackups: 1,
	}
	fmt.Println(config)
}
