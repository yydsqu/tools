package log

import (
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"
)

func TestTrace(t *testing.T) {
	Trace("Trace", "msg", "SSS")
	Debug("Debug", "msg", "SSS")
	Info("Info", "msg", "SSS")
	Warn("Warn", "msg", "SSS")
	Error("Error", "msg", "SSS")
}

func Format(title string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "bg")
	fmt.Fprintln(w, "0\thttp://127.0.0.1:1000\t329923029\t3.885185ms")
	fmt.Fprintln(w, "1\thttp://127.0.0.1:5000\t329923029\t14.101237ms")
	fmt.Fprintln(w, "2\thttp://127.0.0.1:3000\t329923029\t14.936766ms")
	fmt.Fprintln(w, "3\thttp://127.0.0.1:2000\t452\t164.020695ms")
	fmt.Fprintln(w, "4\thttps://chatgpt.com/c/67e7c074-594c\t1\t0ms")
	w.Flush()
}

/*
============================rpc info===============================
# |                             node |        slot |        delay |
0 |            http://127.0.0.1:1000 |   329923029 |   3.885185ms |
1 |            http://127.0.0.1:5000 |   329923029 |  14.101237ms |
2 |            http://127.0.0.1:3000 |   329923029 |  14.936766ms |
3 |            http://127.0.0.1:2000 |   329923028 | 164.020695ms |
4 |            http://127.0.0.1:4000 |   329923025 |  21.755991ms |
===================================================================
*/

func TestMetric(t *testing.T) {
	metric := &Metric{
		Level:          LevelTrace,
		Name:           "测试",
		NotifyPeriod:   time.Second * 5,
		EvaluatePeriod: time.Second * 2,
		Threshold:      50,
	}
	l := NewLogger(NewMetricHandler(NewTerminalHandlerWithLevel(os.Stdout, LevelWarn, true), metric))
	for i := 0; i < 100; i++ {
		l.Error("Error", "msg", i)
	}
}

func TestName(t *testing.T) {
	Format("rpc info", [][]string{
		{"node", "slot", "delay"},
		{"http://127.0.0.1:1000", "329923029", "3.885185ms"},
	})
}
