package system

import (
	"testing"

	"summit/pkg/model"
	"summit/pkg/test"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferSystemState(t *testing.T) {
	// Setup mock filesystem
	AppFs = afero.NewMemMapFs()

	// Setup /etc/apk/world
	require.NoError(t, AppFs.MkdirAll("/etc/apk", 0755))
	require.NoError(t, afero.WriteFile(AppFs, "/etc/apk/world", []byte("package1\npackage2\n"), 0644))

	// Setup /etc/init.d
	require.NoError(t, AppFs.MkdirAll("/etc/init.d", 0755))
	require.NoError(t, afero.WriteFile(AppFs, "/etc/init.d/service1", []byte("#!/bin/sh"), 0755))

	// Note: Skipping service enablement setup as afero mem fs doesn't support symlinks

	// Setup /etc/passwd
	require.NoError(t, afero.WriteFile(AppFs, "/etc/passwd", []byte("root:x:0:0:root:/root:/bin/bash\ntestuser:x:1000:1000:testuser:/home/testuser:/bin/bash\n"), 0644))

	// Setup /etc/group
	require.NoError(t, afero.WriteFile(AppFs, "/etc/group", []byte("root:x:0:\ntestuser:x:1000:\nwheel:x:10:\n"), 0644))

	// Mock runner for apk audit and groups
	runner := test.NewMockCommandRunner()
	runner.SetResponse("", "apk audit", []byte("A /etc/test.conf"))
	runner.SetResponse("", "groups testuser", []byte("testuser wheel"))

	// Setup /etc/test.conf
	require.NoError(t, afero.WriteFile(AppFs, "/etc/test.conf", []byte("content"), 0644))

	state, _, err := InferSystemState(runner, false)
	require.NoError(t, err)

	// Check packages
	assert.Len(t, state.Packages, 2)
	assert.Contains(t, state.Packages, model.PackageState{Name: "package1"})
	assert.Contains(t, state.Packages, model.PackageState{Name: "package2"})

	// Check users
	assert.Len(t, state.Users, 1)
	assert.Equal(t, "testuser", state.Users[0].Name)
	assert.Equal(t, "testuser", state.Users[0].PrimaryGroup)
	assert.Contains(t, state.Users[0].Groups, "wheel")

	// Check configs
	assert.Len(t, state.Configs, 1)
	assert.Equal(t, "/etc/test.conf", state.Configs[0].Path)
	assert.Equal(t, model.OriginUserCreated, state.Configs[0].Origin)
}

func TestInferSystemState_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(afero.Fs)
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing apk world file",
			setupFunc: func(fs afero.Fs) {
				// Don't create /etc/apk/world
				fs.MkdirAll("/etc/init.d", 0755)
				afero.WriteFile(fs, "/etc/init.d/service", []byte("script"), 0755)
				afero.WriteFile(fs, "/etc/passwd", []byte("user:x:1000:1000::/home/user:/bin/bash"), 0644)
			},
			expectError: true,
			errorMsg:    "Error reading /etc/apk/world",
		},
		{
			name: "missing init.d directory",
			setupFunc: func(fs afero.Fs) {
				fs.MkdirAll("/etc/apk", 0755)
				afero.WriteFile(fs, "/etc/apk/world", []byte("pkg1"), 0644)
				// Don't create /etc/init.d
				afero.WriteFile(fs, "/etc/passwd", []byte("user:x:1000:1000::/home/user:/bin/bash"), 0644)
			},
			expectError: true,
			errorMsg:    "error reading /etc/init.d",
		},
		{
			name: "missing passwd file",
			setupFunc: func(fs afero.Fs) {
				fs.MkdirAll("/etc/apk", 0755)
				afero.WriteFile(fs, "/etc/apk/world", []byte("pkg1"), 0644)
				fs.MkdirAll("/etc/init.d", 0755)
				afero.WriteFile(fs, "/etc/group", []byte(""), 0644)
				// Don't create /etc/passwd
			},
			expectError: true,
			errorMsg:    "Error opening /etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			AppFs = afero.NewMemMapFs()
			if tt.setupFunc != nil {
				tt.setupFunc(AppFs)
			}

			runner := test.NewMockCommandRunner()

			// Execute
			_, _, err := InferSystemState(runner, false)

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

func TestInferSystemState_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(afero.Fs, *test.MockCommandRunner)
		validate  func(t *testing.T, state *model.SystemState)
	}{
		{
			name: "empty apk world",
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				fs.MkdirAll("/etc/apk", 0755)
				afero.WriteFile(fs, "/etc/apk/world", []byte(""), 0644)
				fs.MkdirAll("/etc/init.d", 0755)
				afero.WriteFile(fs, "/etc/passwd", []byte(""), 0644)
				afero.WriteFile(fs, "/etc/group", []byte(""), 0644)
			},
			validate: func(t *testing.T, state *model.SystemState) {
				assert.Empty(t, state.Packages)
				assert.Empty(t, state.Services)
				assert.Empty(t, state.Users)
			},
		},
		{
			name: "apk world with empty lines",
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				fs.MkdirAll("/etc/apk", 0755)
				afero.WriteFile(fs, "/etc/apk/world", []byte("pkg1\n\npkg2\n"), 0644)
				fs.MkdirAll("/etc/init.d", 0755)
				afero.WriteFile(fs, "/etc/passwd", []byte(""), 0644)
				afero.WriteFile(fs, "/etc/group", []byte(""), 0644)
			},
			validate: func(t *testing.T, state *model.SystemState) {
				assert.Len(t, state.Packages, 2)
				assert.Equal(t, "pkg1", state.Packages[0].Name)
				assert.Equal(t, "pkg2", state.Packages[1].Name)
			},
		},
		{
			name: "users with different UIDs",
			setupFunc: func(fs afero.Fs, runner *test.MockCommandRunner) {
				fs.MkdirAll("/etc/apk", 0755)
				afero.WriteFile(fs, "/etc/apk/world", []byte(""), 0644)
				fs.MkdirAll("/etc/init.d", 0755)
				afero.WriteFile(fs, "/etc/passwd", []byte("root:x:0:0:root:/root:/bin/bash\nuser1:x:1000:1000:user1:/home/user1:/bin/bash\nuser2:x:999:999:user2:/home/user2:/bin/bash\n"), 0644)
				afero.WriteFile(fs, "/etc/group", []byte("user1:x:1000:\nuser2:x:999:\nwheel:x:10:\n"), 0644)
				runner.SetResponse("", "groups user1", []byte("user1 wheel"))
				runner.SetResponse("", "groups user2", []byte("user2"))
			},
			validate: func(t *testing.T, state *model.SystemState) {
				assert.Len(t, state.Users, 1) // Only user1 (UID >= 1000)
				assert.Equal(t, "user1", state.Users[0].Name)
				assert.Contains(t, state.Users[0].Groups, "wheel")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			AppFs = afero.NewMemMapFs()
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(AppFs, runner)
			}

			// Execute
			state, _, err := InferSystemState(runner, false)
			require.NoError(t, err)

			// Validate
			if tt.validate != nil {
				tt.validate(t, state)
			}
		})
	}
}
