package test

import (
	"bytes"
	"fmt"
	"log/slog"

	"summit/pkg/log"
)

// MockCommandRunner is a shared mock implementation of runner.CommandRunner for testing.
// It tracks executed commands and allows setting up responses and errors.
type MockCommandRunner struct {
	Commands     []string            // Track executed commands
	Responses    map[string][]byte   // Response by command key (user:command)
	Errors       map[string]error    // Error by command key
	UserCommands map[string][]string // Track commands by user
}

// NewMockCommandRunner creates a new MockCommandRunner with initialized maps.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Commands:     []string{},
		Responses:    make(map[string][]byte),
		Errors:       make(map[string]error),
		UserCommands: make(map[string][]string),
	}
}

// Run simulates running a command and returns configured response or error.
func (r *MockCommandRunner) Run(user, command string) ([]byte, error) {
	key := user + ":" + command
	r.Commands = append(r.Commands, command)
	if r.UserCommands[user] == nil {
		r.UserCommands[user] = []string{}
	}
	r.UserCommands[user] = append(r.UserCommands[user], command)

	if err, ok := r.Errors[key]; ok {
		return nil, err
	}
	if resp, ok := r.Responses[key]; ok {
		return resp, nil
	}
	return nil, nil
}

// SetResponse configures a response for a specific user:command.
func (r *MockCommandRunner) SetResponse(user, command string, response []byte) {
	r.Responses[user+":"+command] = response
}

// SetError configures an error for a specific user:command.
func (r *MockCommandRunner) SetError(user, command string, err error) {
	r.Errors[user+":"+command] = err
}

// Reset clears all tracked commands and configurations.
func (r *MockCommandRunner) Reset() {
	r.Commands = []string{}
	r.UserCommands = make(map[string][]string)
	r.Responses = make(map[string][]byte)
	r.Errors = make(map[string]error)
}

// MockLogger is a shared mock implementation of Logger for testing.
// It captures logged messages for verification.
type MockLogger struct {
	Messages []string
	Level    slog.Level
}

// NewMockLogger creates a new MockLogger with the specified level.
func NewMockLogger(level slog.Level) *MockLogger {
	return &MockLogger{
		Messages: []string{},
		Level:    level,
	}
}

// Debug captures debug messages.
func (l *MockLogger) Debug(msg string, args ...any) {
	if l.Level <= slog.LevelDebug {
		l.captureMessage("DEBUG", msg, args...)
	}
}

// Info captures info messages.
func (l *MockLogger) Info(msg string, args ...any) {
	if l.Level <= slog.LevelInfo {
		l.captureMessage("INFO", msg, args...)
	}
}

// Warn captures warn messages.
func (l *MockLogger) Warn(msg string, args ...any) {
	if l.Level <= slog.LevelWarn {
		l.captureMessage("WARN", msg, args...)
	}
}

// Error captures error messages.
func (l *MockLogger) Error(msg string, args ...any) {
	if l.Level <= slog.LevelError {
		l.captureMessage("ERROR", msg, args...)
	}
}

func (l *MockLogger) captureMessage(level, msg string, args ...any) {
	// Simple string formatting for captured messages
	buf := &bytes.Buffer{}
	buf.WriteString(level)
	buf.WriteString(": ")
	buf.WriteString(msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			buf.WriteString(" ")
			buf.WriteString(args[i].(string))
			buf.WriteString("=")
			buf.WriteString(fmt.Sprintf("%v", args[i+1]))
		}
	}
	l.Messages = append(l.Messages, buf.String())
}

// Reset clears all captured messages.
func (l *MockLogger) Reset() {
	l.Messages = []string{}
}

// HasMessage checks if any captured message contains the given substring.
func (l *MockLogger) HasMessage(substring string) bool {
	for _, msg := range l.Messages {
		if bytes.Contains([]byte(msg), []byte(substring)) {
			return true
		}
	}
	return false
}

// SlogLogger creates a real slog logger for testing (alternative to mock).
func SlogLogger(level slog.Level) log.Logger {
	buf := &bytes.Buffer{}
	return log.NewSlogLogger(level, buf)
}
