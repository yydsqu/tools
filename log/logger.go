package log

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const errorKey = "LOG_ERROR"

type Level slog.Level

const (
	LevelTrace Level = -8
	LevelDebug Level = Level(slog.LevelDebug)
	LevelInfo  Level = Level(slog.LevelInfo)
	LevelWarn  Level = Level(slog.LevelWarn)
	LevelError Level = Level(slog.LevelError)
	LevelFatal Level = 12
)

func (l Level) String() string {
	str := func(base string, val Level) string {
		if val == 0 {
			return base
		}
		return fmt.Sprintf("%s%+d", base, val)
	}
	switch {
	case l < LevelDebug:
		return str("TRACE", l-LevelDebug)
	case l < LevelInfo:
		return str("DEBUG", l-LevelDebug)
	case l < LevelWarn:
		return str("INFO", l-LevelInfo)
	case l < LevelError:
		return str("WARN", l-LevelWarn)
	case l < LevelFatal:
		return str("ERROR", l-LevelWarn)
	default:
		return str("FATAL", l-LevelError)
	}
}

func (l Level) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, l.String()), nil
}

func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return l.parse(s)
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(data []byte) error {
	return l.parse(string(data))
}

func (l Level) Level() slog.Level {
	return slog.Level(l)
}

func (l *Level) parse(s string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("slog: level string %q: %w", s, err)
		}
	}()

	name := s
	offset := 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		offset, err = strconv.Atoi(s[i:])
		if err != nil {
			return err
		}
	}
	switch strings.ToUpper(name) {
	case "TRACE":
		*l = LevelDebug
	case "DEBUG":
		*l = LevelDebug
	case "INFO":
		*l = LevelInfo
	case "WARN":
		*l = LevelWarn
	case "ERROR":
		*l = LevelError
	case "FATAL":
		*l = LevelError
	default:
		return errors.New("unknown name")
	}
	*l += Level(offset)
	return nil
}

func LevelAlignedString(l slog.Level) string {
	switch Level(l) {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO "
	case LevelWarn:
		return "WARN "
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "unknown level"
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
	l.Write(-8, msg, ctx...)
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
	l.Write(12, msg, ctx...)
	os.Exit(1)
}
