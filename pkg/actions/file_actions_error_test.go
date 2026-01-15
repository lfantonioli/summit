package actions

import (
	"errors"
	"testing"

	"summit/pkg/system"
	"summit/pkg/test"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileCreateAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *FileCreateAction
		setupFunc   func(afero.Fs, *test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid mode string",
			action: &FileCreateAction{
				Path:    "/test/file.txt",
				Content: "content",
				Mode:    "invalid",
			},
			expectError: true,
			errorMsg:    "invalid syntax",
		},
		{
			name: "invalid owner",
			action: &FileCreateAction{
				Path:    "/test/file.txt",
				Content: "content",
				Owner:   "nonexistentuser",
			},
			expectError: true,
			errorMsg:    "unknown user",
		},
		{
			name: "invalid group",
			action: &FileCreateAction{
				Path:    "/test/file.txt",
				Content: "content",
				Group:   "nonexistentgroup",
			},
			expectError: true,
			errorMsg:    "unknown group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			system.AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(system.AppFs, runner)
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

func TestFileUpdateAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *FileUpdateAction
		setupFunc   func(afero.Fs, *test.MockCommandRunner)
		expectError bool
	}{
		{
			name: "file does not exist",
			action: &FileUpdateAction{
				Path:       "/nonexistent/file.txt",
				NewContent: "new content",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			system.AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(system.AppFs, runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestFileDeleteAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *FileDeleteAction
		setupFunc   func(afero.Fs, *test.MockCommandRunner)
		expectError bool
	}{
		{
			name: "file does not exist",
			action: &FileDeleteAction{
				Path: "/nonexistent/file.txt",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			system.AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(system.AppFs, runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestFileChmodAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *FileChmodAction
		setupFunc   func(afero.Fs, *test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid mode string",
			action: &FileChmodAction{
				Path: "/test/file.txt",
				Mode: "invalid",
			},
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				afero.WriteFile(fs, "/test/file.txt", []byte("content"), 0644)
			},
			expectError: true,
			errorMsg:    "invalid syntax",
		},
		{
			name: "file does not exist",
			action: &FileChmodAction{
				Path: "/nonexistent/file.txt",
				Mode: "0755",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			system.AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(system.AppFs, runner)
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

func TestFileRevertAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *FileRevertAction
		setupFunc   func(afero.Fs, *test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "cached apk not found",
			action: &FileRevertAction{
				Path:         "/etc/test.conf",
				OwnerPackage: "testpkg",
			},
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				runner.SetResponse("", "apk info testpkg", []byte("testpkg-1.0-r0 description:"))
				// Create the file so it can be read
				afero.WriteFile(fs, "/etc/test.conf", []byte("modified content"), 0644)
			},
			expectError: true,
			errorMsg:    "cached apk not found",
		},
		{
			name: "apk info command fails",
			action: &FileRevertAction{
				Path:         "/etc/test.conf",
				OwnerPackage: "testpkg",
			},
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				runner.SetError("", "apk info testpkg", errors.New("package not found"))
			},
			expectError: true,
		},
		{
			name: "tar extraction fails",
			action: &FileRevertAction{
				Path:         "/etc/test.conf",
				OwnerPackage: "testpkg",
			},
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				runner.SetResponse("", "apk info testpkg", []byte("testpkg-1.0-r0 description:"))
				runner.SetError("", "tar -xzf /var/cache/apk/testpkg-1.0-r0.apk -C /tmp/summit-apk- /etc/test.conf", errors.New("tar failed"))
				afero.WriteFile(fs, "/etc/test.conf", []byte("modified content"), 0644)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			system.AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(system.AppFs, runner)
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
