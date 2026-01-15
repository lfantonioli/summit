package actions

import (
	"summit/pkg/log"
	"summit/pkg/system"
)

// Action represents a single, discrete change to the system.
type Action interface {
	// Description returns a human-readable string of what the action does.
	Description() string
	// Apply executes the action.
	Apply(runner system.CommandRunner, logger log.Logger) error
	// Rollback undoes the action. It must be able to restore the system
	// to the state it was in before Apply() was called.
	Rollback(runner system.CommandRunner, logger log.Logger) error
	// ExecutionDetails returns a slice of strings describing the low-level operations.
	ExecutionDetails() []string
}
