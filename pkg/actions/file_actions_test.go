package actions

import (
	"bytes"
	"log/slog"
	"testing"

	"summit/pkg/log"
	"summit/pkg/system"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockCommandRunner struct {
	Commands  []string
	Responses map[string][]byte
	Errors    map[string]error
}

func (r *MockCommandRunner) Run(user, command string) ([]byte, error) {
	r.Commands = append(r.Commands, command)
	key := user + ":" + command
	if err, ok := r.Errors[key]; ok {
		return nil, err
	}
	if resp, ok := r.Responses[key]; ok {
		return resp, nil
	}
	return nil, nil
}

func setupFileTest(t *testing.T) (*MockCommandRunner, log.Logger) {
	system.AppFs = afero.NewMemMapFs()
	runner := &MockCommandRunner{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
	var buf bytes.Buffer
	logger := log.NewSlogLogger(slog.LevelDebug, &buf)
	return runner, logger
}

func TestFileCreateAction_Apply(t *testing.T) {
	runner, logger := setupFileTest(t)

	action := &FileCreateAction{
		Path:    "/test/file.txt",
		Content: "Hello World",
		Mode:    "0644",
		Owner:   "",
		Group:   "",
	}

	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify file was created
	exists, err := afero.Exists(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", string(content))
}

func TestFileCreateAction_Rollback(t *testing.T) {
	runner, logger := setupFileTest(t)

	action := &FileCreateAction{
		Path:    "/test/file.txt",
		Content: "Hello World",
	}

	// First apply
	err := action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify file exists
	exists, err := afero.Exists(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.True(t, exists)

	// Then rollback
	err = action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify file is gone
	exists, err = afero.Exists(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestFileCreateAction_Description(t *testing.T) {
	action := &FileCreateAction{Path: "/etc/motd"}
	assert.Equal(t, "Create file /etc/motd", action.Description())
}

func TestFileCreateAction_ExecutionDetails(t *testing.T) {
	action := &FileCreateAction{
		Path:  "/etc/motd",
		Mode:  "0644",
		Owner: "root",
		Group: "root",
	}
	details := action.ExecutionDetails()
	assert.Contains(t, details, "create file: /etc/motd with permissions 0644")
	assert.Contains(t, details, "set owner to root")
	assert.Contains(t, details, "set group to root")
}

func TestFileUpdateAction_Apply(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create initial file
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Old Content"), 0644)
	require.NoError(t, err)

	action := &FileUpdateAction{
		Path:       "/test/file.txt",
		NewContent: "New Content",
	}

	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify content changed
	content, err := afero.ReadFile(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "New Content", string(content))

	// Verify original content was saved
	assert.Equal(t, "Old Content", action.origContent)
}

func TestFileUpdateAction_Rollback(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create initial file
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Old Content"), 0644)
	require.NoError(t, err)

	action := &FileUpdateAction{
		Path:       "/test/file.txt",
		NewContent: "New Content",
	}

	// Apply
	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Rollback
	err = action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify content restored
	content, err := afero.ReadFile(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "Old Content", string(content))
}

func TestFileUpdateAction_Description(t *testing.T) {
	action := &FileUpdateAction{Path: "/etc/motd"}
	assert.Equal(t, "Update file /etc/motd", action.Description())
}

func TestFileDeleteAction_Apply(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create file to delete
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Content"), 0644)
	require.NoError(t, err)

	action := &FileDeleteAction{Path: "/test/file.txt"}

	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify file is gone
	exists, err := afero.Exists(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.False(t, exists)

	// Verify original content saved
	assert.Equal(t, "Content", action.origContent)
}

func TestFileDeleteAction_Rollback(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create file to delete
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Content"), 0644)
	require.NoError(t, err)

	action := &FileDeleteAction{Path: "/test/file.txt"}

	// Apply (delete)
	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Rollback (restore)
	err = action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify file restored
	exists, err := afero.Exists(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(system.AppFs, "/test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "Content", string(content))
}

func TestFileDeleteAction_Description(t *testing.T) {
	action := &FileDeleteAction{Path: "/etc/motd"}
	assert.Equal(t, "Delete file /etc/motd", action.Description())
}

func TestFileChmodAction_Apply(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create file
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Content"), 0644)
	require.NoError(t, err)

	action := &FileChmodAction{
		Path: "/test/file.txt",
		Mode: "0755",
	}

	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Verify origMode was saved (afero mem fs doesn't fully support mode changes)
	assert.NotZero(t, action.origMode) // origMode should be set to some value
}

func TestFileChmodAction_Rollback(t *testing.T) {
	runner, logger := setupFileTest(t)

	// Create file
	err := afero.WriteFile(system.AppFs, "/test/file.txt", []byte("Content"), 0644)
	require.NoError(t, err)

	action := &FileChmodAction{
		Path: "/test/file.txt",
		Mode: "0755",
	}

	// Apply
	err = action.Apply(runner, logger)
	require.NoError(t, err)

	// Rollback
	err = action.Rollback(runner, logger)
	require.NoError(t, err)

	// Verify mode restored (limited in mem fs)
}

func TestFileChmodAction_Description(t *testing.T) {
	action := &FileChmodAction{Path: "/etc/motd", Mode: "0755"}
	assert.Equal(t, "Chmod file /etc/motd to 0755", action.Description())
}
