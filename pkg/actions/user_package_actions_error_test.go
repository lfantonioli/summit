package actions

import (
	"errors"
	"testing"

	"summit/pkg/model"
	"summit/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPackageAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      UserPackageAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty user",
			action: UserPackageAction{
				User:    "",
				Manager: "pipx",
				Package: "black",
				State:   model.PackageStatePresent,
			},
			expectError: true,
		},
		{
			name: "empty manager",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "",
				Package: "black",
				State:   model.PackageStatePresent,
			},
			expectError: true,
		},
		{
			name: "empty package",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "",
				State:   model.PackageStatePresent,
			},
			expectError: true,
		},
		{
			name: "invalid state",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "black",
				State:   "invalid",
			},
			expectError: true,
			errorMsg:    "unknown user package state",
		},
		{
			name: "unsupported manager",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "unsupported-manager",
				Package: "black",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "unsupported-manager install black", errors.New("unsupported-manager: command not found"))
			},
			expectError: true,
			errorMsg:    "command not found",
		},
		{
			name: "user does not exist",
			action: UserPackageAction{
				User:    "nonexistentuser",
				Manager: "pipx",
				Package: "black",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("nonexistentuser", "pipx install black", errors.New("su: user nonexistentuser does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "package not found",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "nonexistent-package",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "pipx install nonexistent-package", errors.New("pipx: package 'nonexistent-package' not found"))
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "package already installed",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "black",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "pipx install black", errors.New("pipx: package 'black' is already installed"))
			},
			expectError: true,
			errorMsg:    "already installed",
		},
		{
			name: "package not installed for removal",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "black",
				State:   model.PackageStateAbsent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "pipx uninstall black", errors.New("pipx: package 'black' is not installed"))
			},
			expectError: true,
			errorMsg:    "not installed",
		},
		{
			name: "permission denied",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "npm",
				Package: "typescript",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "npm install typescript", errors.New("npm: EACCES: permission denied"))
			},
			expectError: true,
			errorMsg:    "permission denied",
		},
		{
			name: "package name with special characters",
			action: UserPackageAction{
				User:    "testuser",
				Manager: "pipx",
				Package: "package with spaces",
				State:   model.PackageStatePresent,
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("testuser", "pipx install package with spaces", errors.New("pipx: invalid package name"))
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
