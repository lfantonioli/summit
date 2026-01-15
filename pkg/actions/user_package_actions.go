package actions

import (
	"fmt"
	"strings"
	"summit/pkg/log"
	"summit/pkg/model"
	"summit/pkg/system"
)

type UserPackageAction struct {
	User    string
	Manager string // "pipx", "npm"
	Package string
	State   model.UserPackageActionState // "present" or "absent"
}

func (a UserPackageAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.User) == "" {
		return fmt.Errorf("user cannot be empty")
	}
	if strings.TrimSpace(a.Manager) == "" {
		return fmt.Errorf("manager cannot be empty")
	}
	if strings.TrimSpace(a.Package) == "" {
		return fmt.Errorf("package cannot be empty")
	}

	var command string

	switch a.State {
	case model.PackageStatePresent:
		command = fmt.Sprintf("%s install %s", a.Manager, a.Package)
	case model.PackageStateAbsent:
		command = fmt.Sprintf("%s uninstall %s", a.Manager, a.Package)
	default:
		return fmt.Errorf("unknown user package state: %s", a.State)
	}

	logger.Info("Running user package command", "user", a.User, "manager", a.Manager, "command", command)
	_, err := runner.Run(a.User, command)
	return err
}

func (a UserPackageAction) Ptr() Action {
	return &a
}

func (a UserPackageAction) Description() string {
	return fmt.Sprintf("Ensure user package '%s' for user '%s' managed by '%s' is %s", a.Package, a.User, a.Manager, a.State)
}

func (a UserPackageAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	// For user packages, the rollback is the opposite action.
	// This is a simplification; a more robust implementation might store the previous state.
	oppositeState := model.PackageStatePresent
	if a.State == model.PackageStatePresent {
		oppositeState = model.PackageStateAbsent
	}

	oppositeAction := UserPackageAction{
		User:    a.User,
		Manager: a.Manager,
		Package: a.Package,
		State:   oppositeState,
	}

	err := oppositeAction.Apply(runner, logger)
	if err != nil {
		// This is a rollback, so we log the error.
		logger.Error("Failed to roll back user package action", "user", a.User, "package", a.Package, "error", err)
		logger.Warn("The user's package environment may be in an inconsistent state and may require manual intervention.", "manager", a.Manager)
	}
	return err
}

func (a UserPackageAction) ExecutionDetails() []string {
	verb := "install"
	if a.State == model.PackageStateAbsent {
		verb = "uninstall"
	}
	command := fmt.Sprintf("su -l %s -c '%s %s %s'", a.User, a.Manager, verb, a.Package)
	return []string{command}
}
