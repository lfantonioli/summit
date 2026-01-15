package log

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestSlogLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelDebug, &buf)

	logger.Debug("test debug", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test debug")
	assert.Contains(t, output, "key=value")
}

func TestSlogLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelInfo, &buf)

	logger.Info("test info", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test info")
	assert.Contains(t, output, "key=value")
}

func TestSlogLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelWarn, &buf)

	logger.Warn("test warn", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test warn")
	assert.Contains(t, output, "key=value")
}

func TestSlogLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelError, &buf)

	logger.Error("test error", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test error")
	assert.Contains(t, output, "key=value")
}

func TestSlogLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewSlogLogger(slog.LevelWarn, &buf)

	logger.Debug("debug message") // should be filtered out
	logger.Info("info message")   // should be filtered out
	logger.Warn("warn message")   // should appear
	logger.Error("error message") // should appear

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}
