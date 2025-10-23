package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		level Level
		name  string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			SetLevel(tt.level)
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"invalid", LevelInfo}, // Default
		{"", LevelInfo},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	SetOutput(&buf, LevelDebug)

	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")

	output := buf.String()

	// Check that all messages were logged
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message not logged")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message not logged")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message not logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message not logged")
	}

	// Check that key-value pairs are included
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Error("Key-value pairs not logged")
	}
}

func TestLevelFiltering(t *testing.T) {
	// Set level to INFO, debug messages should not appear
	var buf bytes.Buffer
	SetOutput(&buf, LevelInfo)

	Debug("debug message")
	Info("info message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not be logged at INFO level")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message should be logged at INFO level")
	}
}
