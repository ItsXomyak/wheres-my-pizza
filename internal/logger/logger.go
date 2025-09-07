package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime/debug"
	"time"
)

type Logger struct {
	service  string
	hostname string
	handler  *slog.Logger
}

func NewLogger(service string) *Logger {
	hostname, _ := os.Hostname()

	handler := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &Logger{
		service:  service,
		hostname: hostname,
		handler:  handler,
	}
}

func (l *Logger) Info(action, requestID, message string) {
	l.handler.LogAttrs(
		context.TODO(),
		slog.LevelInfo,
		message,
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("service", l.service),
		slog.String("hostname", l.hostname),
		slog.String("action", action),
		slog.String("request_id", requestID),
	)
}

func (l *Logger) Debug(action, requestID, message string) {
	l.handler.LogAttrs(
		context.TODO(),
		slog.LevelDebug,
		message,
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("service", l.service),
		slog.String("hostname", l.hostname),
		slog.String("action", action),
		slog.String("request_id", requestID),
	)
}

func (l *Logger) Error(action, requestID, message string, err error) {
	l.handler.LogAttrs(
		context.TODO(),
		slog.LevelError,
		message,
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		slog.String("service", l.service),
		slog.String("hostname", l.hostname),
		slog.String("action", action),
		slog.String("request_id", requestID),
		slog.Group("error",
			slog.String("msg", err.Error()),
			slog.String("stack", string(debug.Stack())),
		),
	)
}
