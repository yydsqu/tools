package log

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

const errorKey = "LOG_ERROR"

type Logger interface {
	New(ctx ...any) Logger

	Handler() slog.Handler

	With(ctx ...any) Logger

	WithGroup(name string) Logger

	Enabled(ctx context.Context, level slog.Level) bool

	Log(level slog.Level, msg string, ctx ...any)

	Trace(msg string, ctx ...any)

	Debug(msg string, ctx ...any)

	Info(msg string, ctx ...any)

	Warn(msg string, ctx ...any)

	Error(msg string, ctx ...any)

	Fatal(msg string, ctx ...any)

	Write(level slog.Level, msg string, attrs ...any)
}

type Log struct {
	inner  *slog.Logger
	writer io.WriteCloser
}

func (l *Log) New(ctx ...any) Logger {
	return l.With(ctx...)
}

func (l *Log) Handler() slog.Handler {
	return l.inner.Handler()
}

func (l *Log) Write(level slog.Level, msg string, attrs ...any) {
	if !l.inner.Enabled(context.Background(), level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	if len(attrs)%2 != 0 {
		attrs = append(attrs, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(attrs...)
	l.inner.Handler().Handle(context.Background(), r)
}

func (l *Log) Log(level slog.Level, msg string, attrs ...any) {
	l.Write(level, msg, attrs...)
}

func (l *Log) With(ctx ...any) Logger {
	return &Log{
		inner:  l.inner.With(ctx...),
		writer: l.writer,
	}
}

func (l *Log) WithGroup(name string) Logger {
	return &Log{
		inner:  l.inner.WithGroup(name),
		writer: l.writer,
	}
}

func (l *Log) Enabled(ctx context.Context, level slog.Level) bool {
	return l.inner.Enabled(ctx, level)
}

func (l *Log) Trace(msg string, ctx ...any) {
	l.Write(-8, msg, ctx...)
}

func (l *Log) Debug(msg string, ctx ...any) {
	l.Write(slog.LevelDebug, msg, ctx...)
}

func (l *Log) Info(msg string, ctx ...any) {
	l.Write(slog.LevelInfo, msg, ctx...)
}

func (l *Log) Warn(msg string, ctx ...any) {
	l.Write(slog.LevelWarn, msg, ctx...)
}

func (l *Log) Error(msg string, ctx ...any) {
	l.Write(slog.LevelError, msg, ctx...)
}

func (l *Log) Fatal(msg string, ctx ...any) {
	l.Write(12, msg, ctx...)
	os.Exit(1)
}

func (l *Log) Close() {
	if l.writer != nil && l.writer != os.Stdout && l.writer != os.Stderr {
		l.writer.Close()
	}
}

func NewLogger(useColor bool, level Level, output string, maxBackup, maxSize int) *Log {
	var (
		writer io.WriteCloser
		err    error
	)
	switch strings.ToLower(output) {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		useColor = false
		if writer, err = NewAsyncFileWriter(output, cmp.Or(maxSize, 512), cmp.Or(maxBackup, 15)); err != nil {
			fmt.Fprintf(os.Stderr, "flush and close file error. err=%s", err)
			os.Exit(0)
		}
	}

	return &Log{
		inner:  slog.New(NewTerminalHandlerWithLevel(writer, level, useColor)),
		writer: writer,
	}
}

func NewLoggerWithConfig(conf *Config) *Log {
	var (
		writer io.WriteCloser
		err    error
	)
	switch strings.ToLower(conf.Output) {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		if writer, err = NewAsyncFileWriter(conf.Output, cmp.Or(conf.MaxSize, 512), cmp.Or(conf.MaxBackups, 15)); err != nil {
			fmt.Fprintf(os.Stderr, "flush and close file error. err=%s", err)
			os.Exit(0)
		}
	}
	var handler slog.Handler = NewTerminalHandlerWithLevel(writer, conf.Level, conf.UseColor)
	if conf.Prometheus {
		handler = NewPrometheusHandler(conf.AppName, handler)
	}
	return &Log{
		inner:  slog.New(handler),
		writer: writer,
	}
}
