package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
)

// ErrorDetails holds structured error information
type ErrorDetails struct {
	Msg   string `json:"msg"`
	Stack string `json:"stack"`
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Service   string                 `json:"service"`
	Action    string                 `json:"action"`
	Message   string                 `json:"message"`
	Hostname  string                 `json:"hostname"`
	RequestID string                 `json:"request_id"`
	Error     *ErrorDetails          `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Logger provides structured logging functionality
type Logger struct {
	service  string
	hostname string
}

// New creates a new logger instance
func New(service string) *Logger {
	hostname, _ := os.Hostname()
	return &Logger{
		service:  service,
		hostname: hostname,
	}
}

// Debug logs a debug level message
func (l *Logger) Debug(action, message, requestID string, details map[string]interface{}) {
	l.log(DEBUG, action, message, requestID, details, nil)
}

// Info logs an info level message
func (l *Logger) Info(action, message, requestID string, details map[string]interface{}) {
	l.log(INFO, action, message, requestID, details, nil)
}

// Error logs an error level message with error details
func (l *Logger) Error(action, message, requestID string, err error, details map[string]interface{}) {
	var errorDetails *ErrorDetails
	if err != nil {
		stack := l.getStackTrace()
		errorDetails = &ErrorDetails{
			Msg:   err.Error(),
			Stack: stack,
		}
	}
	l.log(ERROR, action, message, requestID, details, errorDetails)
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, action, message, requestID string, details map[string]interface{}, errorDetails *ErrorDetails) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Action:    action,
		Message:   message,
		Hostname:  l.hostname,
		RequestID: requestID,
		Error:     errorDetails,
		Details:   details,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	fmt.Println(string(data))
}

// getStackTrace returns a stack trace for error logging
func (l *Logger) getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	
	var stack string
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		stack += fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
	return stack
}

// GenerateRequestID generates a unique request ID
func GenerateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
