package actions

import (
	"bytes"
	"log/slog"
	"testing"

	"summit/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserTest(t *testing.T) (*MockCommandRunner, log.Logger) {
	runner := &MockCommandRunner{
		Commands:  []string{},
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
	var buf bytes.Buffer
	logger := log.NewSlogLogger(slog.LevelDebug, &buf)
	return runner, logger
}

func TestUserCreateAction_Apply(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &UserCreateAction{UserName: "testuser"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "adduser -D testuser")
}

func TestUserCreateAction_Rollback(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &UserCreateAction{UserName: "testuser"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "deluser testuser")
}

func TestUserCreateAction_Description(t *testing.T) {
	action := &UserCreateAction{UserName: "testuser"}
	assert.Equal(t, "Create user testuser", action.Description())
}

func TestUserCreateAction_ExecutionDetails(t *testing.T) {
	action := &UserCreateAction{UserName: "testuser"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: adduser -D testuser"}, details)
}

func TestUserRemoveAction_Apply(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &UserRemoveAction{UserName: "testuser"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "deluser testuser")
}

func TestUserRemoveAction_Rollback(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &UserRemoveAction{UserName: "testuser"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "adduser -D testuser")
}

func TestUserRemoveAction_Description(t *testing.T) {
	action := &UserRemoveAction{UserName: "testuser"}
	assert.Equal(t, "Remove user testuser", action.Description())
}

func TestUserRemoveAction_ExecutionDetails(t *testing.T) {
	action := &UserRemoveAction{UserName: "testuser"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: deluser testuser"}, details)
}

func TestGroupCreateAction_Apply(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &GroupCreateAction{GroupName: "testgroup"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "addgroup testgroup")
}

func TestGroupCreateAction_Rollback(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &GroupCreateAction{GroupName: "testgroup"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "delgroup testgroup")
}

func TestGroupCreateAction_Description(t *testing.T) {
	action := &GroupCreateAction{GroupName: "testgroup"}
	assert.Equal(t, "Create group testgroup", action.Description())
}

func TestGroupCreateAction_ExecutionDetails(t *testing.T) {
	action := &GroupCreateAction{GroupName: "testgroup"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: addgroup testgroup"}, details)
}

func TestAddUserToGroupAction_Apply(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &AddUserToGroupAction{UserName: "testuser", GroupName: "testgroup"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "addgroup testuser testgroup")
}

func TestAddUserToGroupAction_Rollback(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &AddUserToGroupAction{UserName: "testuser", GroupName: "testgroup"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "delgroup testuser testgroup")
}

func TestAddUserToGroupAction_Description(t *testing.T) {
	action := &AddUserToGroupAction{UserName: "testuser", GroupName: "testgroup"}
	assert.Equal(t, "Add user testuser to group testgroup", action.Description())
}

func TestAddUserToGroupAction_ExecutionDetails(t *testing.T) {
	action := &AddUserToGroupAction{UserName: "testuser", GroupName: "testgroup"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: addgroup testuser testgroup"}, details)
}

func TestRemoveUserFromGroupAction_Apply(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &RemoveUserFromGroupAction{UserName: "testuser", GroupName: "testgroup"}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "delgroup testuser testgroup")
}

func TestRemoveUserFromGroupAction_Rollback(t *testing.T) {
	runner, logger := setupUserTest(t)

	action := &RemoveUserFromGroupAction{UserName: "testuser", GroupName: "testgroup"}

	err := action.Rollback(runner, logger)
	require.NoError(t, err)

	assert.Contains(t, runner.Commands, "addgroup testuser testgroup")
}

func TestRemoveUserFromGroupAction_Description(t *testing.T) {
	action := &RemoveUserFromGroupAction{UserName: "testuser", GroupName: "testgroup"}
	assert.Equal(t, "Remove user testuser from group testgroup", action.Description())
}

func TestRemoveUserFromGroupAction_ExecutionDetails(t *testing.T) {
	action := &RemoveUserFromGroupAction{UserName: "testuser", GroupName: "testgroup"}
	details := action.ExecutionDetails()
	assert.Equal(t, []string{"run: delgroup testuser testgroup"}, details)
}
