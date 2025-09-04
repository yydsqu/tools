package log

import (
	"testing"
)

func TestTrace(t *testing.T) {
	Debug("", "信息", `""事变拉开中国人民抗日战争的序幕''`)
	Info("", "信息", "事变拉开中国人民抗日战争的序幕")
	Warn("事变拉开中国人民抗日战争的序幕", "信息", "事变拉开中国人民抗日战争的序幕")
	Error("事变拉开中国人", "信息", "sssss")
}
