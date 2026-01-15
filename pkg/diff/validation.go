package diff

import (
	"fmt"
	"strings"
	"summit/pkg/model"
)

// ValidationError holds a list of dependency errors
type ValidationError struct {
	errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("dependency validation failed:\n  - %s", strings.Join(e.errors, "\n  - "))
}

// ValidateDependencies checks for missing dependencies in the desired state
func ValidateDependencies(desired *model.SystemState, current *model.SystemState) error {
	var errors []string

	errors = append(errors, validateUserPackageDependencies(desired)...)
	errors = append(errors, validateServiceDependencies(desired, current)...)
	errors = append(errors, validateUserDependencies(desired, current)...)

	if len(errors) > 0 {
		return &ValidationError{errors: errors}
	}

	return nil
}

func validateUserPackageDependencies(desired *model.SystemState) []string {
	var errors []string

	desiredSystemPackages := make(map[string]bool)
	for _, p := range desired.Packages {
		desiredSystemPackages[p.Name] = true
	}

	pipxPackages := []string{}
	npmPackages := []string{}

	for _, userPackage := range desired.UserPackages {
		if len(userPackage.Pipx) > 0 {
			pipxPackages = append(pipxPackages, userPackage.Pipx...)
		}
		if len(userPackage.Npm) > 0 {
			npmPackages = append(npmPackages, userPackage.Npm...)
		}
	}

	if len(pipxPackages) > 0 && !desiredSystemPackages["pipx"] {
		errors = append(errors, fmt.Sprintf("user packages require 'pipx' to be installed for packages: %s. Add 'pipx' to the system packages list.", strings.Join(pipxPackages, ", ")))
	}
	if len(npmPackages) > 0 && !desiredSystemPackages["npm"] {
		errors = append(errors, fmt.Sprintf("user packages require 'npm' to be installed for packages: %s. Add 'npm' to the system packages list.", strings.Join(npmPackages, ", ")))
	}

	return errors
}

func validateServiceDependencies(desired *model.SystemState, current *model.SystemState) []string {
	var errors []string

	availableServices := make(map[string]bool)
	for _, s := range current.Services {
		availableServices[s.Name] = true
	}

	for _, s := range desired.Services {
		if !availableServices[s.Name] {
			errors = append(errors, fmt.Sprintf("service '%s' not found", s.Name))
		}
	}

	return errors
}

func validateUserDependencies(desired *model.SystemState, current *model.SystemState) []string {
	var errors []string

	currentUserMap := make(map[string]bool)
	for _, u := range current.Users {
		currentUserMap[u.Name] = true
	}

	for _, up := range desired.UserPackages {
		if !currentUserMap[up.User] {
			errors = append(errors, fmt.Sprintf("user '%s' not found for user-packages", up.User))
		}
	}

	return errors
}
