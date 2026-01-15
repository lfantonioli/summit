package actions

import (
	"fmt"
	"strings"
	"summit/pkg/log"
	"summit/pkg/system"
)

// ServiceEnableAction enables and starts a service.
type ServiceEnableAction struct {
	ServiceName string
	Runlevel    string
}

func (a *ServiceEnableAction) Description() string {
	return fmt.Sprintf("Enable and start service %s in runlevel %s", a.ServiceName, a.Runlevel)
}

func (a *ServiceEnableAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.ServiceName) == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if strings.TrimSpace(a.Runlevel) == "" {
		return fmt.Errorf("runlevel cannot be empty")
	}
	logger.Info("Enabling and starting service", "service", a.ServiceName, "runlevel", a.Runlevel)
	if _, err := runner.Run("", fmt.Sprintf("rc-update add %s %s", a.ServiceName, a.Runlevel)); err != nil {
		return err
	}
	_, err := runner.Run("", fmt.Sprintf("rc-service %s start", a.ServiceName))
	return err
}

func (a *ServiceEnableAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Stopping and disabling service during rollback", "service", a.ServiceName)
	var lastErr error
	if _, err := runner.Run("", fmt.Sprintf("rc-service %s stop", a.ServiceName)); err != nil {
		logger.Error("Failed to stop service during rollback", "service", a.ServiceName, "error", err)
		lastErr = err
	}
	if _, err := runner.Run("", fmt.Sprintf("rc-update del %s %s", a.ServiceName, a.Runlevel)); err != nil {
		logger.Error("Failed to disable service during rollback", "service", a.ServiceName, "error", err)
		lastErr = err
	}
	return lastErr
}

func (a *ServiceEnableAction) ExecutionDetails() []string {
	return []string{
		fmt.Sprintf("run: rc-update add %s %s", a.ServiceName, a.Runlevel),
		fmt.Sprintf("run: rc-service %s start", a.ServiceName),
	}
}

// ServiceDisableAction stops and disables a service.
type ServiceDisableAction struct {
	ServiceName string
	Runlevel    string
}

func (a *ServiceDisableAction) Description() string {
	return fmt.Sprintf("Stop and disable service %s in runlevel %s", a.ServiceName, a.Runlevel)
}

func (a *ServiceDisableAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.ServiceName) == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if strings.TrimSpace(a.Runlevel) == "" {
		return fmt.Errorf("runlevel cannot be empty")
	}
	logger.Info("Stopping and disabling service", "service", a.ServiceName, "runlevel", a.Runlevel)
	if _, err := runner.Run("", fmt.Sprintf("rc-service %s stop", a.ServiceName)); err != nil {
		return err
	}
	_, err := runner.Run("", fmt.Sprintf("rc-update del %s %s", a.ServiceName, a.Runlevel))
	return err
}

func (a *ServiceDisableAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Enabling and starting service during rollback", "service", a.ServiceName)
	var lastErr error
	if _, err := runner.Run("", fmt.Sprintf("rc-update add %s %s", a.ServiceName, a.Runlevel)); err != nil {
		logger.Error("Failed to enable service during rollback", "service", a.ServiceName, "error", err)
		lastErr = err
	}
	if _, err := runner.Run("", fmt.Sprintf("rc-service %s start", a.ServiceName)); err != nil {
		logger.Error("Failed to start service during rollback", "service", a.ServiceName, "error", err)
		lastErr = err
	}
	return lastErr
}

func (a *ServiceDisableAction) ExecutionDetails() []string {
	return []string{
		fmt.Sprintf("run: rc-service %s stop", a.ServiceName),
		fmt.Sprintf("run: rc-update del %s %s", a.ServiceName, a.Runlevel),
	}
}
