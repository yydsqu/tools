package log

import (
	"testing"
)

func TestNewAsyncFileWriter(t *testing.T) {
	config := Config{
		Output:     "./logs/log.log",
		UseColor:   false,
		MaxBackups: 1,
	}
	log, err := config.Logger()
	if err != nil {
		t.Fatal(err)
	}
	defer config.asyncFileWriter.Stop()
	log.Info("11111111111111111111")
	config.asyncFileWriter.timeTimer.Reset(0)
	log.Info("33333333333333333333333333")

}
