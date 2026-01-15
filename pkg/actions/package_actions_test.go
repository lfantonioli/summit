package actions

import (
	"bytes"
	"log/slog"
	"testing"

	"summit/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPackageTest(t *testing.T) (*MockCommandRunner, log.Logger) {
	runner := &MockCommandRunner{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
	var buf bytes.Buffer
	logger := log.NewSlogLogger(slog.LevelDebug, &buf)
	return runner, logger
}

func TestPackageInstallAction_Apply(t *testing.T) {
	runner, logger := setupPackageTest(t)

	action := &PackageInstallAction{PackageName: "htop"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify command was run
	assert.Contains(t, runner.Commands, "apk add htop")
}

func TestPackageInstallAction_Rollback(t *testing.T) {
	runner, logger := setupPackageTest(t)

	action := &PackageInstallAction{PackageName: "htop"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify rollback command was run
	assert.Contains(t, runner.Commands, "apk del htop")
}

func TestPackageInstallAction_Description(t *testing.T) {
	action := &PackageInstallAction{PackageName: "htop"}
	assert.Equal(t, "Install package htop", action.Description())
}

func TestPackageInstallAction_ExecutionDetails(t *testing.T) {
	action := &PackageInstallAction{PackageName: "htop"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: apk add htop"}, details)
}

func TestPackageRemoveAction_Apply(t *testing.T) {
	runner, logger := setupPackageTest(t)

	action := &PackageRemoveAction{PackageName: "htop"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify command was run
	assert.Contains(t, runner.Commands, "apk del htop")
}

func TestPackageRemoveAction_Rollback(t *testing.T) {
	runner, logger := setupPackageTest(t)

	action := &PackageRemoveAction{PackageName: "htop"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify rollback command was run
	assert.Contains(t, runner.Commands, "apk add htop")
}

func TestPackageRemoveAction_Description(t *testing.T) {
	action := &PackageRemoveAction{PackageName: "htop"}
	assert.Equal(t, "Remove package htop", action.Description())
}

func TestPackageRemoveAction_ExecutionDetails(t *testing.T) {
	action := &PackageRemoveAction{PackageName: "htop"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: apk del htop"}, details)
}
