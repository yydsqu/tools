package log

import (
	"fmt"
	"testing"
	"time"
)

func TestNewAsyncFileWriter(t *testing.T) {
	now := time.Now()
	for i := 0; i < 72; i++ {
		de := now.Add(time.Duration(i) * time.Hour)
		fmt.Println(nextAlignedTime(de, 1))
	}
}
