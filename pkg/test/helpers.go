package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// SetupTestFilesystem creates a temporary directory and returns an afero filesystem.
// The caller is responsible for setting system.AppFs if needed.
// Returns the filesystem and a cleanup function that should be deferred.
func SetupTestFilesystem(t *testing.T) (afero.Fs, func()) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "summit-test-")
	require.NoError(t, err)

	// Set up afero filesystem
	fs := afero.NewBasePathFs(afero.NewOsFs(), tempDir)

	// Return cleanup function
	return fs, func() {
		os.RemoveAll(tempDir)
	}
}

// SetupMockFilesystem creates an in-memory filesystem for testing.
// The caller is responsible for setting system.AppFs if needed.
// Returns the filesystem for direct manipulation.
func SetupMockFilesystem(t *testing.T) afero.Fs {
	return afero.NewMemMapFs()
}

// CreateTestFile creates a file with content in the test filesystem.
func CreateTestFile(t *testing.T, fs afero.Fs, path, content string) {
	err := fs.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, path, []byte(content), 0644)
	require.NoError(t, err)
}

// CreateTestDir creates a directory in the test filesystem.
func CreateTestDir(t *testing.T, fs afero.Fs, path string) {
	err := fs.MkdirAll(path, 0755)
	require.NoError(t, err)
}

// AssertFileExists checks that a file exists and has expected content.
func AssertFileExists(t *testing.T, fs afero.Fs, path, expectedContent string) {
	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)
	require.True(t, exists, "File %s should exist", path)

	if expectedContent != "" {
		content, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		require.Equal(t, expectedContent, string(content))
	}
}

// AssertFileNotExists checks that a file does not exist.
func AssertFileNotExists(t *testing.T, fs afero.Fs, path string) {
	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)
	require.False(t, exists, "File %s should not exist", path)
}

// AssertCommandExecuted checks that a command was executed by the mock runner.
func AssertCommandExecuted(t *testing.T, runner *MockCommandRunner, command string) {
	require.Contains(t, runner.Commands, command, "Command should have been executed: %s", command)
}

// AssertCommandNotExecuted checks that a command was not executed.
func AssertCommandNotExecuted(t *testing.T, runner *MockCommandRunner, command string) {
	require.NotContains(t, runner.Commands, command, "Command should not have been executed: %s", command)
}

// AssertLogContains checks that the logger captured a message containing the substring.
func AssertLogContains(t *testing.T, logger *MockLogger, substring string) {
	require.True(t, logger.HasMessage(substring), "Log should contain: %s", substring)
}

// SetupBasicSystemFiles creates basic system files for testing system inference.
func SetupBasicSystemFiles(t *testing.T, fs afero.Fs) {
	// Create /etc/apk/world
	CreateTestFile(t, fs, "/etc/apk/world", "htop\nvim\n")

	// Create /etc/init.d/service
	CreateTestFile(t, fs, "/etc/init.d/nginx", "#!/bin/sh\necho 'nginx service'")

	// Create /etc/passwd
	CreateTestFile(t, fs, "/etc/passwd", "root:x:0:0:root:/root:/bin/bash\ntestuser:x:1000:1000:testuser:/home/testuser:/bin/bash\n")

	// Create /etc/motd
	CreateTestFile(t, fs, "/etc/motd", "Welcome to Alpine Linux")
}

// SetupMockRunnerForSystemInference configures a mock runner with typical system responses.
func SetupMockRunnerForSystemInference(runner *MockCommandRunner) {
	runner.SetResponse("", "apk audit", []byte("A /etc/motd"))
	runner.SetResponse("", "groups testuser", []byte("testuser wheel"))
}
