package logger

import (
	"encoding/json"
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
	Output     string `json:"output" toml:"output"`
	MaxSize    int    `json:"max_bytes_size" toml:"max_bytes_size"`
	MaxBackups int    `json:"max_backups" toml:"max_backups"`
	Level      Level  `json:"level" toml:"level"`
	UseColor   bool   `json:"use_color" toml:"use_color"`
}
