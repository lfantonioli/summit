package actions

import (
	"errors"
	"testing"

	"summit/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageInstallAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *PackageInstallAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty package name",
			action: &PackageInstallAction{
				PackageName: "",
			},
			expectError: true,
		},
		{
			name: "package not found",
			action: &PackageInstallAction{
				PackageName: "nonexistent-package",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk add nonexistent-package", errors.New("ERROR: unable to select packages"))
			},
			expectError: true,
			errorMsg:    "unable to select packages",
		},
		{
			name: "network error",
			action: &PackageInstallAction{
				PackageName: "htop",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk add htop", errors.New("ERROR: http://dl-cdn.alpinelinux.org/alpine/edge/main: network is unreachable"))
			},
			expectError: true,
			errorMsg:    "network is unreachable",
		},
		{
			name: "package name with special characters",
			action: &PackageInstallAction{
				PackageName: "package with spaces",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk add package with spaces", errors.New("ERROR: invalid package name"))
			},
			expectError: true,
		},
		{
			name: "very long package name",
			action: &PackageInstallAction{
				PackageName: "very-long-package-name-that-might-cause-issues-with-command-line-length-limits-or-other-system-constraints-in-the-package-manager",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk add very-long-package-name-that-might-cause-issues-with-command-line-length-limits-or-other-system-constraints-in-the-package-manager", errors.New("command line too long"))
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

func TestPackageRemoveAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *PackageRemoveAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty package name",
			action: &PackageRemoveAction{
				PackageName: "",
			},
			expectError: true,
		},
		{
			name: "package not installed",
			action: &PackageRemoveAction{
				PackageName: "not-installed-package",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk del not-installed-package", errors.New("ERROR: package not-installed-package is not installed"))
			},
			expectError: true,
			errorMsg:    "not installed",
		},
		{
			name: "package is required by others",
			action: &PackageRemoveAction{
				PackageName: "busybox",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk del busybox", errors.New("ERROR: busybox is required by other packages"))
			},
			expectError: true,
			errorMsg:    "required by other packages",
		},
		{
			name: "package name with special characters",
			action: &PackageRemoveAction{
				PackageName: "package@version",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "apk del package@version", errors.New("ERROR: invalid package name"))
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
