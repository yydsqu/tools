package log

import (
	"bytes"
	"fmt"
	"log/slog"
	"math/big"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	timeFormat        = "2006-01-02T15:04:05-0700"
	floatFormat       = 'f'
	termMsgJust       = 40
	termCtxMaxPadding = 40
)

var spaces = []byte("                                        ")

type TerminalStringer interface {
	TerminalString() string
}

func (t *TerminalHandler) format(buf []byte, r slog.Record, useColor bool) []byte {
	msg := escapeMessage(r.Message)
	var color = ""

	if useColor {
		switch Level(r.Level) {
		case LevelFatal:
			color = "\x1b[35m"
		case LevelError:
			color = "\x1b[31m"
		case LevelWarn:
			color = "\x1b[33m"
		case LevelInfo:
			color = "\x1b[32m"
		case LevelDebug:
			color = "\x1b[36m"
		case LevelTrace:
			color = "\x1b[34m"
		}
	}

	if buf == nil {
		buf = make([]byte, 0, 30+termMsgJust)
	}

	b := bytes.NewBuffer(buf)

	if color != "" {
		b.WriteString(color)
		b.WriteString(LevelAlignedString(r.Level))
		b.WriteString("\x1b[0m")
	} else {
		b.WriteString(LevelAlignedString(r.Level))
	}
	b.WriteString("[")
	writeTimeTermFormat(b, r.Time)
	b.WriteString("] ")
	b.WriteString(msg)
	length := len(msg)

	if (r.NumAttrs()+len(t.attrs)) > 0 && length < termMsgJust {
		b.Write(spaces[:termMsgJust-length])
	}

	t.formatAttributes(b, r, color)

	return b.Bytes()
}

func (t *TerminalHandler) formatAttributes(buf *bytes.Buffer, r slog.Record, color string) {
	var tmp = make([]byte, 40)
	writeAttr := func(attr slog.Attr, first, last bool) {
		buf.WriteByte(' ')

		if color != "" {
			buf.WriteString(color)
			//buf.Write(appendEscapeString(buf.AvailableBuffer(), attr.Key))
			buf.Write(appendEscapeString(tmp[:0], attr.Key))
			buf.WriteString("\x1b[0m=")
		} else {
			//buf.Write(appendEscapeString(buf.AvailableBuffer(), attr.Key))
			buf.Write(appendEscapeString(tmp[:0], attr.Key))
			buf.WriteByte('=')
		}
		//val := FormatSlogValue(attr.Value, true, buf.AvailableBuffer())
		val := FormatSlogValue(attr.Value, tmp[:0])

		padding := t.fieldPadding[attr.Key]

		length := utf8.RuneCount(val)
		if padding < length && length <= termCtxMaxPadding {
			padding = length
			t.fieldPadding[attr.Key] = padding
		}
		buf.Write(val)
		if !last && padding > length {
			buf.Write(spaces[:padding-length])
		}
	}
	var n = 0
	var nAttrs = len(t.attrs) + r.NumAttrs()
	for _, attr := range t.attrs {
		writeAttr(attr, n == 0, n == nAttrs-1)
		n++
	}
	r.Attrs(func(attr slog.Attr) bool {
		writeAttr(attr, n == 0, n == nAttrs-1)
		n++
		return true
	})
	buf.WriteByte('\n')
}

func FormatSlogValue(v slog.Value, tmp []byte) (result []byte) {
	var value any
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = []byte("<nil>")
			} else {
				panic(err)
			}
		}
	}()

	switch v.Kind() {
	case slog.KindString:
		return appendEscapeString(tmp, v.String())
	case slog.KindInt64:
		return strconv.AppendInt(tmp, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(tmp, v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.AppendFloat(tmp, v.Float64(), floatFormat, 3, 64)
	case slog.KindBool:
		return strconv.AppendBool(tmp, v.Bool())
	case slog.KindDuration:
		value = v.Duration()
	case slog.KindTime:
		return v.Time().AppendFormat(tmp, timeFormat)
	default:
		value = v.Any()
	}
	if value == nil {
		return []byte("<nil>")
	}
	switch v := value.(type) {
	case *big.Int:
		return appendEscapeString(tmp, v.String())
	case error:
		return appendEscapeString(tmp, v.Error())
	case TerminalStringer:
		return appendEscapeString(tmp, v.TerminalString())
	case fmt.Stringer:
		return appendEscapeString(tmp, v.String())
	}

	internal := fmt.Appendf(tmp, "%+v", value)
	return appendEscapeString(tmp, string(internal))
}

func appendEscapeString(dst []byte, s string) []byte {
	needsQuoting := false
	needsEscaping := false
	for _, r := range s {
		if r == ' ' || r == '=' {
			needsQuoting = true
			continue
		}
		if r <= '"' || r > '~' {
			needsEscaping = true
			break
		}
	}
	if needsEscaping {
		return append(dst, []byte(s)...)
	}
	if needsQuoting {
		dst = append(dst, '"')
		dst = append(dst, []byte(s)...)
		return append(dst, '"')
	}
	return append(dst, []byte(s)...)
}

func escapeMessage(s string) string {
	needsQuoting := false
	for _, r := range s {
		if r == '\r' || r == '\n' || r == '\t' {
			continue
		}
		if r < ' ' || r > '~' || r == '=' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return s
}

func writeTimeTermFormat(buf *bytes.Buffer, t time.Time) {
	_, month, day := t.Date()
	writePosIntWidth(buf, int(month), 2)
	buf.WriteByte('-')
	writePosIntWidth(buf, day, 2)
	buf.WriteByte('|')
	hour, m, sec := t.Clock()
	writePosIntWidth(buf, hour, 2)
	buf.WriteByte(':')
	writePosIntWidth(buf, m, 2)
	buf.WriteByte(':')
	writePosIntWidth(buf, sec, 2)
	ns := t.Nanosecond()
	buf.WriteByte('.')
	writePosIntWidth(buf, ns/1e6, 3)
}

func writePosIntWidth(b *bytes.Buffer, i, width int) {
	if i < 0 {
		panic("negative int")
	}
	var bb [20]byte
	bp := len(bb) - 1
	for i >= 10 || width > 1 {
		width--
		q := i / 10
		bb[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	bb[bp] = byte('0' + i)
	b.Write(bb[bp:])
}
