package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var defaultLogger *slog.Logger
var otelSlogHandler = otelslog.NewHandler("java-metadata", otelslog.WithSource(true))

// Level represents a log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func init() {
	// Initialize with INFO level by default
	SetLevel(LevelInfo)
}

// SetLevel configures the global logger with the specified level
func SetLevel(level Level) {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)

	// Wrap with otelslog to add trace context to logs
	defaultLogger = slog.New(slogmulti.Fanout(
		handler,
		otelSlogHandler,
	))
}

// SetOutput configures the logger to write to the specified writer
func SetOutput(w io.Writer, level Level) {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}
	handler := slog.NewTextHandler(w, opts)
	// Wrap with otelslog to add trace context to logs
	defaultLogger = slog.New(slogmulti.Fanout(
		handler,
		otelSlogHandler,
	))
}

// Debug logs a debug message
func Debug(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

// Info logs an informational message
func Info(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message
func Warn(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

// Error logs an error message
func Error(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

// ParseLevel converts a string level to Level
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
