// Package runner defines interfaces for command execution.
// This package exists to break import cycles between testing and system packages.
package runner

// CommandRunner defines an interface for running commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(user, command string) ([]byte, error)
}
