package system

import (
	"os/exec"

	"summit/pkg/runner"
)

// CommandRunner defines an interface for running commands.
// This allows for mocking in tests.
// Re-exported from pkg/runner to maintain backward compatibility.
type CommandRunner = runner.CommandRunner

// LiveCommandRunner is an implementation of CommandRunner that runs commands on the live system.
type LiveCommandRunner struct{}

// Run executes the given command and returns its output.
func (r *LiveCommandRunner) Run(user, command string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	return cmd.CombinedOutput()
}
