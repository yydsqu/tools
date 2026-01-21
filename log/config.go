package log

type Config struct {
	AppName    string `json:"app_name" toml:"app_name"`
	Prometheus bool   `json:"prometheus" toml:"prometheus"`
	UseColor   bool   `json:"use_color" toml:"use_color"`
	Level      Level  `json:"level" toml:"level"`
	Output     string `json:"output" toml:"output"`
	MaxSize    int    `json:"max_bytes_size" toml:"max_bytes_size"`
	MaxBackups int    `json:"max_backups" toml:"max_backups"`
}

func (conf *Config) Logger() *Log {
	return NewLoggerWithConfig(conf)
}
