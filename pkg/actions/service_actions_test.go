package actions

import (
	"bytes"
	"log/slog"
	"testing"

	"summit/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServiceTest(t *testing.T) (*MockCommandRunner, log.Logger) {
	runner := &MockCommandRunner{
		Commands:  []string{},
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
	var buf bytes.Buffer
	logger := log.NewSlogLogger(slog.LevelDebug, &buf)
	return runner, logger
}

func TestServiceEnableAction_Apply(t *testing.T) {
	runner, logger := setupServiceTest(t)

	action := &ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify commands were run
	assert.Contains(t, runner.Commands, "rc-update add nginx default")
	assert.Contains(t, runner.Commands, "rc-service nginx start")
}

func TestServiceEnableAction_Rollback(t *testing.T) {
	runner, logger := setupServiceTest(t)

	action := &ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify rollback commands were run
	assert.Contains(t, runner.Commands, "rc-service nginx stop")
	assert.Contains(t, runner.Commands, "rc-update del nginx default")
}

func TestServiceEnableAction_Description(t *testing.T) {
	action := &ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"}
	assert.Equal(t, "Enable and start service nginx in runlevel default", action.Description())
}

func TestServiceEnableAction_ExecutionDetails(t *testing.T) {
	action := &ServiceEnableAction{ServiceName: "nginx", Runlevel: "default"}
	details := action.ExecutionDetails()
	expected := []string{
		"run: rc-update add nginx default",
		"run: rc-service nginx start",
	}
	assert.Equal(t, expected, details)
}

func TestServiceDisableAction_Apply(t *testing.T) {
	runner, logger := setupServiceTest(t)

	action := &ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify commands were run
	assert.Contains(t, runner.Commands, "rc-service nginx stop")
	assert.Contains(t, runner.Commands, "rc-update del nginx default")
}

func TestServiceDisableAction_Rollback(t *testing.T) {
	runner, logger := setupServiceTest(t)

	action := &ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify rollback commands were run
	assert.Contains(t, runner.Commands, "rc-update add nginx default")
	assert.Contains(t, runner.Commands, "rc-service nginx start")
}

func TestServiceDisableAction_Description(t *testing.T) {
	action := &ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"}
	assert.Equal(t, "Stop and disable service nginx in runlevel default", action.Description())
}

func TestServiceDisableAction_ExecutionDetails(t *testing.T) {
	action := &ServiceDisableAction{ServiceName: "nginx", Runlevel: "default"}
	details := action.ExecutionDetails()
	expected := []string{
		"run: rc-service nginx stop",
		"run: rc-update del nginx default",
	}
	assert.Equal(t, expected, details)
}
