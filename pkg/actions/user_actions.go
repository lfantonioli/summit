package actions

import (
	"fmt"
	"strings"
	"summit/pkg/log"
	"summit/pkg/system"
)

// UserCreateAction creates a user.
type UserCreateAction struct {
	UserName string
}

func (a *UserCreateAction) Description() string {
	return fmt.Sprintf("Create user %s", a.UserName)
}

func (a *UserCreateAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.UserName) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	logger.Info("Creating user", "user", a.UserName)
	_, err := runner.Run("", fmt.Sprintf("adduser -D %s", a.UserName))
	if err != nil {
		return err
	}
	logger.Warn("User created without password", "user", a.UserName, "note", "Set a password with 'chpasswd "+a.UserName+"' if login access is needed")
	return nil
}

func (a *UserCreateAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back user creation", "user", a.UserName)
	_, err := runner.Run("", fmt.Sprintf("deluser %s", a.UserName))
	if err != nil {
		logger.Error("Failed to roll back user creation", "user", a.UserName, "error", err)
	}
	return err
}

func (a *UserCreateAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: adduser -D %s", a.UserName)}
}

// UserRemoveAction removes a user.
type UserRemoveAction struct {
	UserName string
}

func (a *UserRemoveAction) Description() string {
	return fmt.Sprintf("Remove user %s", a.UserName)
}

func (a *UserRemoveAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.UserName) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	logger.Info("Removing user", "user", a.UserName)
	_, err := runner.Run("", fmt.Sprintf("deluser %s", a.UserName))
	return err
}

func (a *UserRemoveAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back user removal", "user", a.UserName)
	_, err := runner.Run("", fmt.Sprintf("adduser -D %s", a.UserName))
	if err != nil {
		logger.Error("Failed to roll back user removal", "user", a.UserName, "error", err)
	}
	return err
}

func (a *UserRemoveAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: deluser %s", a.UserName)}
}

// GroupCreateAction creates a group.
type GroupCreateAction struct {
	GroupName string
}

func (a *GroupCreateAction) Description() string {
	return fmt.Sprintf("Create group %s", a.GroupName)
}

func (a *GroupCreateAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.GroupName) == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	logger.Info("Creating group", "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("addgroup %s", a.GroupName))
	return err
}

func (a *GroupCreateAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back group creation", "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("delgroup %s", a.GroupName))
	if err != nil {
		logger.Error("Failed to roll back group creation", "group", a.GroupName, "error", err)
	}
	return err
}

func (a *GroupCreateAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: addgroup %s", a.GroupName)}
}

// AddUserToGroupAction adds a user to a group.
type AddUserToGroupAction struct {
	UserName  string
	GroupName string
}

func (a *AddUserToGroupAction) Description() string {
	return fmt.Sprintf("Add user %s to group %s", a.UserName, a.GroupName)
}

func (a *AddUserToGroupAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.UserName) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.TrimSpace(a.GroupName) == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	logger.Info("Adding user to group", "user", a.UserName, "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("addgroup %s %s", a.UserName, a.GroupName))
	return err
}

func (a *AddUserToGroupAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back adding user to group", "user", a.UserName, "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("delgroup %s %s", a.UserName, a.GroupName))
	if err != nil {
		logger.Error("Failed to roll back adding user to group", "user", a.UserName, "group", a.GroupName, "error", err)
	}
	return err
}

func (a *AddUserToGroupAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: addgroup %s %s", a.UserName, a.GroupName)}
}

// RemoveUserFromGroupAction removes a user from a group.
type RemoveUserFromGroupAction struct {
	UserName  string
	GroupName string
}

func (a *RemoveUserFromGroupAction) Description() string {
	return fmt.Sprintf("Remove user %s from group %s", a.UserName, a.GroupName)
}

func (a *RemoveUserFromGroupAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.UserName) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.TrimSpace(a.GroupName) == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	logger.Info("Removing user from group", "user", a.UserName, "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("delgroup %s %s", a.UserName, a.GroupName))
	return err
}

func (a *RemoveUserFromGroupAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back removing user from group", "user", a.UserName, "group", a.GroupName)
	_, err := runner.Run("", fmt.Sprintf("addgroup %s %s", a.UserName, a.GroupName))
	if err != nil {
		logger.Error("Failed to roll back removing user from group", "user", a.UserName, "group", a.GroupName, "error", err)
	}
	return err
}

func (a *RemoveUserFromGroupAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: delgroup %s %s", a.UserName, a.GroupName)}
}
