package log

import (
	"github.com/shuiziliu7788/go-tools/notify"
	"os"
	"testing"
	"time"
)

func TestTrace(t *testing.T) {
	Trace("Trace", "msg", "SSS")
	Debug("Debug", "msg", "SSS")
	Info("Info", "msg", "SSS")
	Warn("Warn", "msg", "SSS")
	Error("Error", "msg", "SSS")
}

func TestMetric(t *testing.T) {
	metric := &Metric{
		Level:          LevelTrace,
		Name:           "测试",
		NotifyPeriod:   time.Second * 5,
		EvaluatePeriod: time.Second * 2,
		Threshold:      50,
		Notify: &notify.WxPusher{
			AppToken: "AT_dWk1PSaCmPieZ8MkuY7KqsOHxuwARB3t",
			TopicIds: []int{31179},
			Uids:     nil,
		},
	}
	l := NewLogger(NewMetricHandler(NewTerminalHandlerWithLevel(os.Stdout, LevelWarn, true), metric))
	for i := 0; i < 100; i++ {
		l.Error("Error", "msg", i)
	}
}
