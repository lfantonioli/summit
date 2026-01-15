package actions

import (
	"fmt"
	"strings"
	"summit/pkg/log"
	"summit/pkg/system"
)

// PackageInstallAction installs a package.
type PackageInstallAction struct {
	PackageName string
}

func (a *PackageInstallAction) Description() string {
	return fmt.Sprintf("Install package %s", a.PackageName)
}

func (a *PackageInstallAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.PackageName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	logger.Info("Installing package", "package", a.PackageName)
	_, err := runner.Run("", fmt.Sprintf("apk add %s", a.PackageName))
	return err
}

func (a *PackageInstallAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back package install", "package", a.PackageName)
	_, err := runner.Run("", fmt.Sprintf("apk del %s", a.PackageName))
	if err != nil {
		logger.Error("Failed to roll back package install", "package", a.PackageName, "error", err)
	}
	return err
}

func (a *PackageInstallAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: apk add %s", a.PackageName)}
}

// PackageRemoveAction removes a package.
type PackageRemoveAction struct {
	PackageName string
}

func (a *PackageRemoveAction) Description() string {
	return fmt.Sprintf("Remove package %s", a.PackageName)
}

func (a *PackageRemoveAction) Apply(runner system.CommandRunner, logger log.Logger) error {
	if strings.TrimSpace(a.PackageName) == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	logger.Info("Removing package", "package", a.PackageName)
	_, err := runner.Run("", fmt.Sprintf("apk del %s", a.PackageName))
	return err
}

func (a *PackageRemoveAction) Rollback(runner system.CommandRunner, logger log.Logger) error {
	logger.Info("Rolling back package removal", "package", a.PackageName)
	_, err := runner.Run("", fmt.Sprintf("apk add %s", a.PackageName))
	if err != nil {
		logger.Error("Failed to roll back package removal", "package", a.PackageName, "error", err)
	}
	return err
}

func (a *PackageRemoveAction) ExecutionDetails() []string {
	return []string{fmt.Sprintf("run: apk del %s", a.PackageName)}
}
