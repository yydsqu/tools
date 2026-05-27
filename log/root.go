package log

import (
	"log/slog"
	"os"
	"sync/atomic"
)

var (
	defaultLogger = &Log{
		inner: slog.New(NewTerminalHandler(os.Stdout, true)),
	}
	root atomic.Value
)

func init() {
	SetDefault(defaultLogger)
}

func SetDefault(l Logger) {
	root.Store(l)
	if lg, ok := l.(*Log); ok {
		slog.SetDefault(lg.inner)
	}
}

func Root() Logger {
	return root.Load().(Logger)
}

func Trace(msg string, ctx ...any) {
	Root().Write(slog.Level(LevelTrace), msg, ctx...)
}

func Debug(msg string, ctx ...any) {
	Root().Write(slog.LevelDebug, msg, ctx...)
}

func Info(msg string, ctx ...any) {
	Root().Write(slog.LevelInfo, msg, ctx...)
}

func Warn(msg string, ctx ...any) {
	Root().Write(slog.LevelWarn, msg, ctx...)
}

func Error(msg string, ctx ...any) {
	Root().Write(slog.LevelError, msg, ctx...)
}

func Fatal(msg string, ctx ...any) {
	Root().Write(slog.Level(LevelFatal), msg, ctx...)
	os.Exit(1)
}

func New(ctx ...any) Logger {
	return Root().With(ctx...)
}
