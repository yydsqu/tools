package log

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"
)

const errorKey = "LOG_ERROR"

const (
	legacyLevelFatal = iota
	legacyLevelError
	legacyLevelWarn
	legacyLevelInfo
	legacyLevelDebug
	legacyLevelTrace
)

const (
	LevelTrace slog.Level = -8
	LevelDebug slog.Level = slog.LevelDebug
	LevelInfo  slog.Level = slog.LevelInfo
	LevelWarn  slog.Level = slog.LevelWarn
	LevelError slog.Level = slog.LevelError
	LevelFatal slog.Level = 12
)

func FromLegacyLevel(lvl int) slog.Level {
	switch lvl {
	case legacyLevelFatal:
		return LevelFatal
	case legacyLevelError:
		return slog.LevelError
	case legacyLevelWarn:
		return slog.LevelWarn
	case legacyLevelInfo:
		return slog.LevelInfo
	case legacyLevelDebug:
		return slog.LevelDebug
	case legacyLevelTrace:
		return LevelTrace
	default:
		break
	}

	if lvl > legacyLevelTrace {
		return LevelTrace
	}
	return LevelFatal
}

func LevelAlignedString(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO "
	case slog.LevelWarn:
		return "WARN "
	case slog.LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "unknown level"
	}
}

func LevelString(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "trace"
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "eror"
	case LevelFatal:
		return "crit"
	default:
		return "unknown"
	}
}

type Logger interface {
	With(ctx ...interface{}) Logger

	New(ctx ...interface{}) Logger

	Log(level slog.Level, msg string, ctx ...interface{})

	Trace(msg string, ctx ...interface{})

	Debug(msg string, ctx ...interface{})

	Info(msg string, ctx ...interface{})

	Warn(msg string, ctx ...interface{})

	Error(msg string, ctx ...interface{})

	Fatal(msg string, ctx ...interface{})

	Write(level slog.Level, msg string, attrs ...any)

	Enabled(ctx context.Context, level slog.Level) bool

	Handler() slog.Handler
}

type logger struct {
	inner *slog.Logger
}

func NewLogger(h slog.Handler) Logger {
	return &logger{
		slog.New(h),
	}
}

func (l *logger) Handler() slog.Handler {
	return l.inner.Handler()
}

func (l *logger) Write(level slog.Level, msg string, attrs ...any) {
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

func (l *logger) Log(level slog.Level, msg string, attrs ...any) {
	l.Write(level, msg, attrs...)
}

func (l *logger) With(ctx ...interface{}) Logger {
	return &logger{l.inner.With(ctx...)}
}

func (l *logger) New(ctx ...interface{}) Logger {
	return l.With(ctx...)
}

func (l *logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.inner.Enabled(ctx, level)
}

func (l *logger) Trace(msg string, ctx ...interface{}) {
	l.Write(LevelTrace, msg, ctx...)
}

func (l *logger) Debug(msg string, ctx ...interface{}) {
	l.Write(slog.LevelDebug, msg, ctx...)
}

func (l *logger) Info(msg string, ctx ...interface{}) {
	l.Write(slog.LevelInfo, msg, ctx...)
}

func (l *logger) Warn(msg string, ctx ...any) {
	l.Write(slog.LevelWarn, msg, ctx...)
}

func (l *logger) Error(msg string, ctx ...interface{}) {
	l.Write(slog.LevelError, msg, ctx...)
}

func (l *logger) Fatal(msg string, ctx ...interface{}) {
	l.Write(LevelFatal, msg, ctx...)
	os.Exit(1)
}
