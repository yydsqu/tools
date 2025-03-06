package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

type TerminalHandler struct {
	mu       sync.Mutex
	wr       io.Writer
	lvl      slog.Level
	useColor bool
	attrs    []slog.Attr

	fieldPadding map[string]int

	buf []byte
}

func (t *TerminalHandler) Handle(_ context.Context, r slog.Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	buf := t.format(t.buf, r, t.useColor)
	t.wr.Write(buf)
	t.buf = buf[:0]
	return nil
}

func (t *TerminalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= t.lvl
}

func (t *TerminalHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

func (t *TerminalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TerminalHandler{
		wr:           t.wr,
		lvl:          t.lvl,
		useColor:     t.useColor,
		attrs:        append(t.attrs, attrs...),
		fieldPadding: make(map[string]int),
	}
}

func (t *TerminalHandler) ResetFieldPadding() {
	t.mu.Lock()
	t.fieldPadding = make(map[string]int)
	t.mu.Unlock()
}

func NewTerminalHandler(wr io.Writer, useColor bool) *TerminalHandler {
	return NewTerminalHandlerWithLevel(wr, LevelTrace, useColor)
}

func NewTerminalHandlerWithLevel(wr io.Writer, lvl slog.Level, useColor bool) *TerminalHandler {
	return &TerminalHandler{
		wr:           wr,
		lvl:          lvl,
		useColor:     useColor,
		fieldPadding: make(map[string]int),
	}
}
