package log

import (
	"cmp"
	"encoding/json"
	"github.com/yydsqu/tools/notify"
	"io"
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
	Output               string           `json:"output" toml:"output"`
	MaxSize              int64            `json:"max_bytes_size" toml:"max_bytes_size"`
	MaxBackups           int              `json:"max_backups" toml:"max_backups"`
	RotateHours          uint             `json:"rotate_hours" toml:"rotate_hours"`
	Level                Level            `json:"level" toml:"level"`
	UseColor             bool             `json:"use_color" toml:"use_color"`
	NotifyName           string           `json:"notify_name" toml:"notify_name"`
	NotifyLevel          Level            `json:"notify_level" toml:"notify_level"`
	NotifyThreshold      int              `json:"notify_threshold" toml:"notify_threshold"`
	NotifyEvaluatePeriod Duration         `json:"notify_evaluate_period" toml:"notify_evaluate_period"`
	NotifyPeriod         Duration         `json:"notify_period" toml:"notify_period"`
	Notify               *notify.WxPusher `json:"notify,omitempty" toml:"notify"`
	asyncFileWriter      *AsyncFileWriter
}

func (conf *Config) Logger() (Logger, error) {
	var (
		writer  io.Writer = os.Stdout
		handler slog.Handler
		err     error
	)
	if conf.Output != "" {
		if conf.asyncFileWriter, err = NewAsyncFileWriter(conf.Output, cmp.Or(conf.MaxSize, 512), cmp.Or(conf.MaxBackups, 7), conf.RotateHours); err != nil {
			return nil, err
		}
		writer = conf.asyncFileWriter
	}

	handler = NewTerminalHandlerWithLevel(writer, conf.Level, conf.UseColor)
	if conf.Notify != nil {
		handler = NewMetricHandler(handler, &Metric{
			Notify:         conf.Notify,
			Name:           conf.NotifyName,
			Level:          conf.NotifyLevel,
			NotifyPeriod:   time.Duration(conf.NotifyPeriod),
			EvaluatePeriod: time.Duration(conf.NotifyEvaluatePeriod),
			Threshold:      conf.NotifyThreshold,
		})
	}
	return NewLogger(handler), nil
}

func (conf *Config) Stop() error {
	if conf.asyncFileWriter != nil {
		conf.asyncFileWriter.Stop()
	}
	return nil
}
