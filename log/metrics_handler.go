package log

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	DefaultTemplate, _ = template.New("example").Parse(`
<div style="background-color:#111217;margin: 0;padding: 0">
    <div style="background-color:#22252b;margin:0 auto;max-width:600px;min-height: 200px;padding: 20px">
        <div style="text-align: left;border-bottom:1px solid #2f3037;direction:ltr;font-size:0;padding:10px 0;">
            <strong style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;text-align:left;color:#FFFFFF;line-height: 32px;word-break:break-word;">
                {{.Title}}
            </strong>
        </div>

        <div>
            <div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;text-align:left;color:#FFFFFF;line-height: 42px">
                <strong>日志详情</strong>
            </div>
            <div style="background-color:#111217;border:1px solid #2f3037;vertical-align:top;padding:16px;word-break:break-word; white-space: pre-line;">
                <div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1.5;text-align:left;color:#FFFFFF;word-break:break-word;">{{.Message}}</div>
            </div>
        </div>

        <div style="text-align: center;border-top:1px solid #2f3037;direction:ltr;font-size:0;padding:10px 0;">
            <div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1.5;text-align:left;color:#91929e;">
                {{.Time.Format "2006-01-02 15:04:05"}}
            </div>
        </div>
    </div>
</div>
`)
)

type Notify interface {
	Send(title string, content string)
}

type Metric struct {
	Notify         Notify
	Name           string        `json:"name"`
	Level          slog.Level    `json:"level"`
	NotifyPeriod   time.Duration `json:"notify_period"`   // 30分钟内最多通知一次
	EvaluatePeriod time.Duration `json:"evaluate_period"` // 5分钟内统计日志
	Threshold      int           `json:"threshold"`       // 触发通知的日志数量阈值
	lastNotify     time.Time
	nextNotifyTime time.Time
	records        []*slog.Record
	mu             sync.Mutex
}

func (m *Metric) sendNotification() {
	m.lastNotify = time.Now()
	next := m.lastNotify.Add(m.NotifyPeriod)
	m.lastNotify = next
	if m.Notify == nil {
		return
	}
	index := 0
	if len(m.records) > 15 {
		index = len(m.records) - 15
	}
	var message string
	for _, record := range m.records[index:] {
		message += fmt.Sprintf("%-5s[%s] %-40s\t", record.Level, record.Time.Format("2006-01-02 15:04:05.99"), record.Message)
		record.Attrs(func(attr slog.Attr) bool {
			message += fmt.Sprintf("%s=%v\t", attr.Key, attr.Value)
			return true
		})
		message += "\n"
	}
	builder := &strings.Builder{}
	if err := DefaultTemplate.Execute(builder, map[string]any{
		"Title":   m.Name,
		"Message": message,
		"Time":    time.Now(),
	}); err != nil {
		return
	}
	// 发送通知信息
	m.Notify.Send(fmt.Sprintf("%s[警报]", m.Name), builder.String())
}

func (m *Metric) Handle(record *slog.Record) {
	if record.Level < slog.LevelError { // 只统计 ERROR 及以上的日志
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// 添加日志
	m.records = append(m.records, record)
	cutoffTime := time.Now().Add(-m.EvaluatePeriod)
	idx := sort.Search(len(m.records), func(i int) bool {
		return m.records[i].Time.After(cutoffTime)
	})
	m.records = m.records[idx:]
	if len(m.records) >= m.Threshold && time.Now().After(m.lastNotify) {
		m.sendNotification()
	}
}

type MetricHandler struct {
	handler slog.Handler
	metric  *Metric
}

func (m *MetricHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.handler.Enabled(ctx, level)
}

func (m *MetricHandler) Handle(ctx context.Context, record slog.Record) error {
	m.metric.Handle(&record)
	if m.handler.Enabled(ctx, record.Level) {
		return m.handler.Handle(ctx, record)
	}
	return nil
}

func (m *MetricHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &MetricHandler{
		handler: m.handler.WithAttrs(attrs),
		metric:  m.metric,
	}
}

func (m *MetricHandler) WithGroup(name string) slog.Handler {
	return &MetricHandler{
		handler: m.handler.WithGroup(name),
		metric:  m.metric,
	}
}

func NewMetricHandler(h slog.Handler, metric *Metric) *MetricHandler {
	return &MetricHandler{
		handler: h,
		metric:  metric,
	}
}
