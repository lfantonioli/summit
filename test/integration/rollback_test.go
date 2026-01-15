//go:build integration
// +build integration

package integration

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"summit/pkg/actions"
	"summit/pkg/config"
	"summit/pkg/diff"
	"summit/pkg/log"
	"summit/pkg/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ControlledCommandRunner executes real commands but can inject failures for specific commands
type ControlledCommandRunner struct {
	Errors map[string]error // command -> error to return
}

func NewControlledCommandRunner() *ControlledCommandRunner {
	return &ControlledCommandRunner{
		Errors: make(map[string]error),
	}
}

func (r *ControlledCommandRunner) Run(user, command string) ([]byte, error) {
	// Check if this command should fail
	if err, ok := r.Errors[command]; ok {
		return nil, err
	}

	// Execute the command for real
	cmd := exec.Command("sh", "-c", command)
	return cmd.CombinedOutput()
}

func (r *ControlledCommandRunner) SetError(command string, err error) {
	r.Errors[command] = err
}

func TestMultiActionRollback(t *testing.T) {
	// Test rollback mechanism with multi-action plan including packages, files, and services
	// This simulates a failure during plan execution and verifies all completed actions are rolled back

	runner, plan := setupTestEnvironment(t)
	logger := setupLogger()

	// Capture initial state
	initialFiles := getFileStates(t)

	// Execute actions with controlled failure and rollback
	completedActions := executeActionsWithFailure(t, plan, runner, logger)

	// Verify actions succeeded before failure
	verifyActionsSucceeded(t, completedActions)

	// Verify rollback restored system state
	verifyRollbackSuccess(t, completedActions, initialFiles, runner)

	t.Log("Rollback verification completed successfully")
}

func setupTestEnvironment(t *testing.T) (*ControlledCommandRunner, []actions.Action) {
	// Create necessary directories and service files
	require.NoError(t, os.MkdirAll("/etc/init.d", 0755))
	require.NoError(t, os.MkdirAll("/etc/runlevels/default", 0755))
	require.NoError(t, os.MkdirAll("/etc/runlevels/boot", 0755))

	// Create mock service files
	require.NoError(t, os.WriteFile("/etc/init.d/sshd", []byte("#!/bin/sh\necho sshd service"), 0755))
	require.NoError(t, os.WriteFile("/etc/init.d/crond", []byte("#!/bin/sh\necho crond service"), 0755))

	// Create symlinks to make services appear enabled
	require.NoError(t, os.Symlink("/etc/init.d/sshd", "/etc/runlevels/default/sshd"))
	require.NoError(t, os.Symlink("/etc/init.d/crond", "/etc/runlevels/default/crond"))

	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Load desired config
	configPath := "/app/test/integration/testdata/rollback_test.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	require.NoError(t, err, "Failed to load test config")

	// Setup controlled command runner
	runner := NewControlledCommandRunner()
	runner.SetError("apk add vim", errors.New("simulated package installation failure"))

	// Infer current system state
	current, _, err := system.InferSystemState(runner, false)
	require.NoError(t, err, "Failed to infer system state")

	// Calculate plan
	plan, err := diff.CalculatePlan(desired, current, runner, false)
	require.NoError(t, err, "Failed to calculate plan")
	require.Greater(t, len(plan), 0, "Plan should contain actions")

	return runner, plan
}

func setupLogger() log.Logger {
	var logBuf bytes.Buffer
	return log.NewSlogLogger(slog.LevelDebug, &logBuf)
}

func executeActionsWithFailure(t *testing.T, plan []actions.Action, runner *ControlledCommandRunner, logger log.Logger) []actions.Action {
	completedActions := []actions.Action{}
	failed := false
	var failureErr error

	for _, action := range plan {
		t.Logf("Applying: %s", action.Description())
		if err := action.Apply(runner, logger); err != nil {
			t.Logf("Action failed: %v, initiating rollback", err)
			failed = true
			failureErr = err
			break
		}
		completedActions = append(completedActions, action)
	}

	// Perform rollback if we had a failure
	if failed {
		require.Error(t, failureErr, "Expected the test to have a failure")
		t.Logf("Rolling back %d completed actions", len(completedActions))
		for i := len(completedActions) - 1; i >= 0; i-- {
			action := completedActions[i]
			t.Logf("Rolling back: %s", action.Description())
			if err := action.Rollback(runner, logger); err != nil {
				t.Errorf("Rollback failed for %s: %v", action.Description(), err)
			}
		}
	}

	return completedActions
}

func verifyActionsSucceeded(t *testing.T, completedActions []actions.Action) {
	if len(completedActions) == 0 {
		return
	}

	// Check that files were actually created with correct content and permissions
	for _, action := range completedActions {
		if strings.Contains(action.Description(), "Create file") {
			parts := strings.Split(action.Description(), " ")
			if len(parts) >= 3 {
				filePath := parts[2]
				assert.FileExists(t, filePath, "File %s should exist after successful action", filePath)

				// Verify file has expected content and permissions
				if info, err := os.Stat(filePath); err == nil {
					// Check mode (should be 0644 as specified in config)
					expectedMode := os.FileMode(0644)
					assert.Equal(t, expectedMode, info.Mode().Perm(),
						"File %s should have correct permissions", filePath)

					// Check content for known files
					if content, err := os.ReadFile(filePath); err == nil {
						if filePath == "/etc/rollback_test.conf" {
							assert.Contains(t, string(content), "rollback testing purposes",
								"File %s should contain expected content", filePath)
						} else if filePath == "/tmp/rollback_temp.txt" {
							assert.Equal(t, "Temporary test file", string(content),
								"File %s should contain expected content", filePath)
						}
					}
				}
			}
		}
	}
}

func verifyRollbackSuccess(t *testing.T, completedActions []actions.Action, initialFiles map[string]FileState, runner *ControlledCommandRunner) {
	if len(completedActions) == 0 {
		return
	}

	finalFiles := getFileStates(t)

	// Files should be back to initial state
	for filePath, initialState := range initialFiles {
		finalState, exists := finalFiles[filePath]
		require.True(t, exists, "File %s should be present in final state", filePath)

		assert.Equal(t, initialState.Exists, finalState.Exists,
			"File %s existence should match initial state", filePath)

		if initialState.Exists {
			assert.Equal(t, initialState.Mode, finalState.Mode,
				"File %s permissions should be restored", filePath)
			assert.Equal(t, initialState.Content, finalState.Content,
				"File %s content should be restored", filePath)
		}
	}

	// Check that service symlinks were cleaned up (rollback of service enable)
	for _, action := range completedActions {
		if strings.Contains(action.Description(), "Enable and start service") {
			parts := strings.Split(action.Description(), " ")
			if len(parts) >= 5 {
				serviceName := parts[4]
				runlevel := parts[6]
				symlinkPath := fmt.Sprintf("/etc/runlevels/%s/%s", runlevel, serviceName)
				if _, err := os.Stat(symlinkPath); !os.IsNotExist(err) {
					// If the symlink still exists, it should be the original one, not one created by the test
					assert.Fail(t, "Service symlink %s should not exist after rollback", symlinkPath)
				}
			}
		}
	}
}

// Helper functions for state verification
func getInstalledPackages(t *testing.T, runner *ControlledCommandRunner) map[string]bool {
	cmd := exec.Command("sh", "-c", "apk info | sort")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to get installed packages")

	packages := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages[line] = true
		}
	}
	return packages
}

// FileState captures the complete state of a file for rollback verification
type FileState struct {
	Exists  bool
	Mode    os.FileMode
	Content string
}

func getFileStates(t *testing.T) map[string]FileState {
	files := map[string]string{
		"/etc/rollback_test.conf": "",
		"/tmp/rollback_temp.txt":  "",
	}

	states := make(map[string]FileState)
	for filePath := range files {
		state := FileState{Exists: false, Mode: 0, Content: ""}

		if info, err := os.Stat(filePath); err == nil {
			// File exists
			state.Exists = true
			state.Mode = info.Mode()

			// Read content for text files
			if content, err := os.ReadFile(filePath); err == nil {
				state.Content = string(content)
			} else {
				t.Logf("Warning: Could not read content of %s: %v", filePath, err)
			}
		} else if os.IsNotExist(err) {
			// File doesn't exist - this is expected initial state
			state.Exists = false
		} else {
			t.Fatalf("Failed to stat file %s: %v", filePath, err)
		}

		states[filePath] = state
	}
	return states
}
