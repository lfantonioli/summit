package actions

import (
	"bytes"
	"log/slog"
	"testing"

	"summit/pkg/log"
	"summit/pkg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserPackageTest(t *testing.T) (*MockCommandRunner, log.Logger) {
	runner := &MockCommandRunner{
		Commands:  []string{},
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
	var buf bytes.Buffer
	logger := log.NewSlogLogger(slog.LevelDebug, &buf)
	return runner, logger
}

func TestUserPackageAction_Apply_Present(t *testing.T) {
	runner, logger := setupUserPackageTest(t)

	action := UserPackageAction{
		User:    "testuser",
		Manager: "pipx",
		Package: "black",
		State:   model.PackageStatePresent,
	}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "pipx install black")
	// Note: MockCommandRunner tracks commands, but key is "testuser:pipx install black"
}

func TestUserPackageAction_Apply_Absent(t *testing.T) {
	runner, logger := setupUserPackageTest(t)

	action := UserPackageAction{
		User:    "testuser",
		Manager: "npm",
		Package: "lodash",
		State:   model.PackageStateAbsent,
	}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "npm uninstall lodash")
}

func TestUserPackageAction_Rollback(t *testing.T) {
	runner, logger := setupUserPackageTest(t)

	action := UserPackageAction{
		User:    "testuser",
		Manager: "pipx",
		Package: "black",
		State:   model.PackageStatePresent,
	}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	// Rollback should do the opposite: uninstall
	assert.Contains(t, runner.Commands, "pipx uninstall black")
}

func TestUserPackageAction_Description(t *testing.T) {
	action := UserPackageAction{
		User:    "testuser",
		Manager: "pipx",
		Package: "black",
		State:   model.PackageStatePresent,
	}
	assert.Equal(t, "Ensure user package 'black' for user 'testuser' managed by 'pipx' is present", action.Description())
}

func TestUserPackageAction_ExecutionDetails(t *testing.T) {
	action := UserPackageAction{
		User:    "testuser",
		Manager: "pipx",
		Package: "black",
		State:   model.PackageStatePresent,
	}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"su -l testuser -c 'pipx install black'"}, details)
}

func TestUserPackageAction_ExecutionDetails_Absent(t *testing.T) {
	action := UserPackageAction{
		User:    "testuser",
		Manager: "npm",
		Package: "lodash",
		State:   model.PackageStateAbsent,
	}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"su -l testuser -c 'npm uninstall lodash'"}, details)
}
