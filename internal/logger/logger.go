package logger

import (
	"io"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

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
	defaultLogger = slog.New(handler)
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
	defaultLogger = slog.New(handler)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an informational message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
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
