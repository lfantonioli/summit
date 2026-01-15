package actions

import (
	"errors"
	"testing"

	"summit/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceEnableAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *ServiceEnableAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty service name",
			action: &ServiceEnableAction{
				ServiceName: "",
				Runlevel:    "default",
			},
			expectError: true,
		},
		{
			name: "empty runlevel",
			action: &ServiceEnableAction{
				ServiceName: "nginx",
				Runlevel:    "",
			},
			expectError: true,
		},
		{
			name: "service not found",
			action: &ServiceEnableAction{
				ServiceName: "nonexistent-service",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-update add nonexistent-service default", errors.New("rc-update: service nonexistent-service does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "invalid runlevel",
			action: &ServiceEnableAction{
				ServiceName: "nginx",
				Runlevel:    "invalid-runlevel",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-update add nginx invalid-runlevel", errors.New("rc-update: invalid runlevel 'invalid-runlevel'"))
			},
			expectError: true,
			errorMsg:    "invalid runlevel",
		},
		{
			name: "service already enabled",
			action: &ServiceEnableAction{
				ServiceName: "nginx",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-update add nginx default", errors.New("rc-update: nginx already installed in runlevel default; skipping"))
			},
			expectError: true,
		},
		{
			name: "start command fails",
			action: &ServiceEnableAction{
				ServiceName: "nginx",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetResponse("", "rc-update add nginx default", []byte(""))
				runner.SetError("", "rc-service nginx start", errors.New("rc-service: service nginx failed to start"))
			},
			expectError: true,
			errorMsg:    "failed to start",
		},
		{
			name: "service name with special characters",
			action: &ServiceEnableAction{
				ServiceName: "service with spaces",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-update add service with spaces default", errors.New("invalid service name"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestServiceDisableAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *ServiceDisableAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty service name",
			action: &ServiceDisableAction{
				ServiceName: "",
				Runlevel:    "default",
			},
			expectError: true,
		},
		{
			name: "empty runlevel",
			action: &ServiceDisableAction{
				ServiceName: "nginx",
				Runlevel:    "",
			},
			expectError: true,
		},
		{
			name: "service not enabled",
			action: &ServiceDisableAction{
				ServiceName: "nginx",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-service nginx stop", errors.New("rc-service: service nginx is not running"))
				runner.SetError("", "rc-update del nginx default", errors.New("rc-update: nginx is not installed in runlevel default"))
			},
			expectError: true,
		},
		{
			name: "stop command fails",
			action: &ServiceDisableAction{
				ServiceName: "nginx",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-service nginx stop", errors.New("rc-service: service nginx failed to stop"))
			},
			expectError: true,
			errorMsg:    "failed to stop",
		},
		{
			name: "service name with special characters",
			action: &ServiceDisableAction{
				ServiceName: "service@version",
				Runlevel:    "default",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "rc-service service@version stop", errors.New("invalid service name"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}
