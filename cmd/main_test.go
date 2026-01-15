package cmd

import (
	"bytes"
	"encoding/json"
	"summit/pkg/model"
	"summit/pkg/system"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommandRunner is a mock implementation of the CommandRunner for testing.
type MockCommandRunner struct {
	Commands  []string
	Responses map[string][]byte
	Errors    map[string]error
}

// Run simulates running a command.
func (r *MockCommandRunner) Run(user, command string) ([]byte, error) {
	key := user + ":" + command
	r.Commands = append(r.Commands, key)
	if err, ok := r.Errors[key]; ok {
		return nil, err
	}
	if resp, ok := r.Responses[key]; ok {
		return resp, nil
	}
	return nil, nil
}

func executeCommand(runner *MockCommandRunner, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	cmdRunner = runner

	err := rootCmd.Execute()
	return buf.String(), err
}

func setupTest(t *testing.T) *MockCommandRunner {
	// Set up a mock file system for each test
	system.AppFs = afero.NewMemMapFs()

	// Create some dummy files and directories that are expected to exist
	require.NoError(t, system.AppFs.MkdirAll("/etc/apk", 0755))
	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/apk/world", []byte(""), 0644))
	require.NoError(t, system.AppFs.MkdirAll("/etc/init.d", 0755))
	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/passwd", []byte(""), 0644))
	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/group", []byte(""), 0644))

	return &MockCommandRunner{
		Responses: make(map[string][]byte),
		Errors:    make(map[string]error),
	}
}

func TestApply_CreatesFile(t *testing.T) {
	runner := setupTest(t)
	runner.Responses[":apk audit"] = []byte("")

	config := `
packages:
  - name: htop

configs:
  - path: /etc/motd
    content: |
      Hello from summit!
 `
	require.NoError(t, afero.WriteFile(system.AppFs, "/system.yaml", []byte(config), 0644))

	_, err := executeCommand(runner, "apply", "--config", "/system.yaml")
	require.NoError(t, err)

	// Verify effects: package install command was executed
	assert.Contains(t, runner.Commands, ":apk add htop")

	// Verify the file was created
	content, err := afero.ReadFile(system.AppFs, "/etc/motd")
	require.NoError(t, err)
	assert.Equal(t, "Hello from summit!\n", string(content))
}

func TestDiff_ShowsChanges(t *testing.T) {
	runner := setupTest(t)
	runner.Responses[":apk audit"] = []byte("")

	config := `
packages:
  - name: htop

configs:
  - path: /etc/motd
    content: |
      Hello from summit!
 `
	require.NoError(t, afero.WriteFile(system.AppFs, "/system.yaml", []byte(config), 0644))

	output, err := executeCommand(runner, "diff", "--config", "/system.yaml", "--json")
	require.NoError(t, err)

	// Unmarshal the JSON output and verify the plan
	type actionForJSON struct {
		Type        string
		Description string
	}
	var plan []actionForJSON
	require.NoError(t, json.Unmarshal([]byte(output), &plan))

	assert.Len(t, plan, 2)

	assert.Equal(t, "*actions.PackageInstallAction", plan[0].Type)
	assert.Equal(t, "Install package htop", plan[0].Description)

	assert.Equal(t, "*actions.FileCreateAction", plan[1].Type)
	assert.Equal(t, "Create file /etc/motd", plan[1].Description)

	// Verify no side effects: diff should only run read commands to infer state
	assert.Contains(t, runner.Commands, ":apk audit")
	assert.NotContains(t, runner.Commands, "apk add htop")
}

func TestDump_OutputsSystemState(t *testing.T) {
	runner := setupTest(t)
	runner.Responses[":apk audit"] = []byte("A  /etc/motd")

	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/apk/world", []byte("htop\n"), 0644))
	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/motd", []byte("Hello from summit!"), 0644))

	output, err := executeCommand(runner, "dump", "--json")
	require.NoError(t, err)

	// Unmarshal the JSON output and verify the state
	var state model.SystemState
	require.NoError(t, json.Unmarshal([]byte(output), &state))

	assert.Len(t, state.Packages, 1)
	assert.Equal(t, "htop", state.Packages[0].Name)

	assert.Len(t, state.Configs, 1)
	assert.Equal(t, "/etc/motd", state.Configs[0].Path)
	assert.Equal(t, "Hello from summit!", state.Configs[0].Content)
}

func TestApply_DryRun(t *testing.T) {
	runner := setupTest(t)
	runner.Responses[":apk audit"] = []byte("")
	runner.Responses[":sh -c 'cat /etc/group'"] = []byte("wheel:x:10:\n")

	config := `
packages:
  - name: htop
`
	require.NoError(t, afero.WriteFile(system.AppFs, "/system.yaml", []byte(config), 0644))

	output, err := executeCommand(runner, "apply", "--config", "/system.yaml", "--dry-run", "--json")
	require.NoError(t, err)

	// Unmarshal the JSON output and verify the plan
	type actionForJSON struct {
		Type        string
		Description string
	}
	var plan []actionForJSON
	require.NoError(t, json.Unmarshal([]byte(output), &plan))

	assert.Len(t, plan, 1)
	assert.Equal(t, "*actions.PackageInstallAction", plan[0].Type)
	assert.Equal(t, "Install package htop", plan[0].Description)

	// Verify that only read-only commands were run
	assert.Equal(t, []string{":apk audit", ":sh -c 'cat /etc/group'"}, runner.Commands)
}

func TestDiff_UserPackages(t *testing.T) {
	runner := setupTest(t)
	// Add a mock user to the system
	require.NoError(t, afero.WriteFile(system.AppFs, "/etc/passwd", []byte("testuser:x:1000:1000:,,,:/home/testuser:/bin/bash"), 0644))

	runner.Responses["apk audit"] = []byte("")
	runner.Responses["testuser:pipx list --json"] = []byte(`{
		"venvs": {
			"black": {
				"metadata": {
					"package": "black"
				}
			}
		}
	}`)

	config := `
packages:
  - name: pipx

users:
  - name: testuser
    groups: []

user-packages:
  - user: testuser
    pipx:
      - ruff
`
	require.NoError(t, afero.WriteFile(system.AppFs, "/system.yaml", []byte(config), 0644))

	output, err := executeCommand(runner, "diff", "--config", "/system.yaml", "--json")
	require.NoError(t, err)

	// Unmarshal the JSON output and verify the plan
	type actionForJSON struct {
		Type        string
		Description string
	}
	var plan []actionForJSON
	require.NoError(t, json.Unmarshal([]byte(output), &plan))

	assert.Len(t, plan, 3)

	// Note: order can be non-deterministic, so check for all actions
	foundPipx := false
	foundRuff := false
	foundBlack := false
	for _, action := range plan {
		if action.Type == "*actions.PackageInstallAction" && action.Description == "Install package pipx" {
			foundPipx = true
		}
		if action.Type == "*actions.UserPackageAction" && action.Description == "Ensure user package 'ruff' for user 'testuser' managed by 'pipx' is present" {
			foundRuff = true
		}
		if action.Type == "*actions.UserPackageAction" && action.Description == "Ensure user package 'black' for user 'testuser' managed by 'pipx' is absent" {
			foundBlack = true
		}
	}
	assert.True(t, foundPipx, "Did not find action to install pipx")
	assert.True(t, foundRuff, "Did not find action to add ruff")
	assert.True(t, foundBlack, "Did not find action to remove black")

	// Verify no side effects: diff should only run read commands to infer state
	assert.Contains(t, runner.Commands, ":apk audit")
	assert.Contains(t, runner.Commands, "testuser:pipx list --json")
}
