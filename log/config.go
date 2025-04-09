package log

import (
	"encoding/json"
	"github.com/yydsqu/tools/notify"
	"log/slog"
	"os"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(bytes []byte) error {
	var dur string
	if err := json.Unmarshal(bytes, &dur); err != nil {
		return err
	}
	duration, err := time.ParseDuration(dur)
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

type Config struct {
	LogLevel                Level            `json:"level"`
	LogUseColor             bool             `json:"use_color"`
	LogNotifyName           string           `json:"notify_name"`
	LogNotifyLevel          Level            `json:"notify_level"`
	LogNotifyThreshold      int              `json:"notify_threshold"`
	LogNotifyEvaluatePeriod Duration         `json:"notify_evaluate_period"`
	LogNotifyPeriod         Duration         `json:"notify_period"`
	LogNotify               *notify.WxPusher `json:"notify,omitempty"`
}

func (conf *Config) LogHandler() (handler slog.Handler) {
	handler = NewTerminalHandlerWithLevel(os.Stdout, conf.LogLevel, conf.LogUseColor)
	if conf.LogNotify != nil {
		handler = NewMetricHandler(handler, &Metric{
			Notify:         conf.LogNotify,
			Name:           conf.LogNotifyName,
			Level:          conf.LogNotifyLevel,
			NotifyPeriod:   time.Duration(conf.LogNotifyPeriod),
			EvaluatePeriod: time.Duration(conf.LogNotifyEvaluatePeriod),
			Threshold:      conf.LogNotifyThreshold,
		})
	}
	return handler
}
