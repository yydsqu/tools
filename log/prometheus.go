package log

import (
	"cmp"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	Hostname, _ = os.Hostname()
)

type PrometheusHandler struct {
	appName string
	root    slog.Handler
	counter *prometheus.CounterVec
}

func (prometheus *PrometheusHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return prometheus.root.Enabled(ctx, l)
}

func (prometheus *PrometheusHandler) Handle(ctx context.Context, record slog.Record) error {
	prometheus.counter.WithLabelValues(prometheus.appName, record.Level.String()).Inc()
	return prometheus.root.Handle(ctx, record)
}

func (prometheus *PrometheusHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return prometheus
	}
	appName := prometheus.appName
	for _, attr := range attrs {
		if attr.Key == "app" {
			appName = strings.Join([]string{prometheus.appName, fmt.Sprintf("%s", attr.Value)}, "_")
			break
		}
	}
	return &PrometheusHandler{
		root:    prometheus.root.WithAttrs(attrs),
		appName: appName,
		counter: prometheus.counter,
	}
}

func (prometheus *PrometheusHandler) WithGroup(groupName string) slog.Handler {
	if groupName == "" {
		return prometheus
	}
	return &PrometheusHandler{
		root:    prometheus.root,
		appName: strings.Join([]string{prometheus.appName, groupName}, "_"),
		counter: prometheus.counter,
	}
}

func ExecutableName() string {
	exePath, _ := os.Executable()
	return filepath.Base(exePath)
}

func NewPrometheusHandler(appName string, handler slog.Handler) *PrometheusHandler {
	var (
		counter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_logs_total",
				Help: "Total number of logs processed, partitioned by level.",
				ConstLabels: map[string]string{
					"hostname": Hostname,
				},
			},
			[]string{"app", "level"},
		)
	)
	prometheus.Register(counter)
	return &PrometheusHandler{
		appName: cmp.Or(appName, ExecutableName()),
		root:    handler,
		counter: counter,
	}
}
