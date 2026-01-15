package diff

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"summit/pkg/actions"
	"summit/pkg/model"
	"summit/pkg/system"
)

const groupFilePath = "/etc/group"

const unmanagedFileWarning = "Warning: unmanaged file found %s (created outside package manager). Consider adding to ignored_configs or use --prune-unmanaged to delete.\n"

// MatchesGlob checks if path matches the glob pattern, with support for **
// Supports recursive ** patterns like /etc/ssh/**/*.pub
// Limitations: Only one ** per pattern, doesn't support ** at start or multiple **
func MatchesGlob(pattern, path string) bool {
	if strings.Contains(pattern, "**") {
		// Simple recursive glob support
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			// Handle * in prefix/suffix
			if strings.HasSuffix(prefix, "/*") {
				prefix = prefix[:len(prefix)-2]
			} else if strings.HasSuffix(prefix, "*") {
				prefix = prefix[:len(prefix)-1]
			}
			if strings.HasPrefix(suffix, "/*") {
				suffix = suffix[2:]
			} else if strings.HasPrefix(suffix, "*") {
				suffix = suffix[1:]
			}
			return strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix)
		}
		return false
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// CalculatePlan generates a list of actions to transform the current state into the desired state.
func CalculatePlan(desired *model.SystemState, current *model.SystemState, runner system.CommandRunner, pruneUnmanaged bool) ([]actions.Action, error) {
	if err := ValidateDependencies(desired, current); err != nil {
		return nil, err
	}

	var plan []actions.Action

	plan = append(plan, calculatePackageActions(desired.Packages, current.Packages)...)
	plan = append(plan, calculateServiceActions(desired.Services, current.Services)...)
	userActions, err := calculateUserActions(desired.Users, current.Users, runner)
	if err != nil {
		return nil, err
	}
	plan = append(plan, userActions...)
	plan = append(plan, calculateConfigActions(desired, current, pruneUnmanaged)...)
	plan = append(plan, calculateUserPackageActions(desired, current, runner)...)

	return plan, nil
}

func calculateUserPackageActions(desired *model.SystemState, current *model.SystemState, runner system.CommandRunner) []actions.Action {
	var a []actions.Action

	for _, userPackage := range desired.UserPackages {
		if len(userPackage.Pipx) > 0 {
			// Discover and compare pipx packages
			a = append(a, compareUserPackages(userPackage.User, "pipx", userPackage.Pipx, runner)...)
		}

		if len(userPackage.Npm) > 0 {
			// Discover and compare npm packages
			a = append(a, compareUserPackages(userPackage.User, "npm", userPackage.Npm, runner)...)
		}
	}

	return a
}

type PipxPackageMetadata struct {
	Package string `json:"package"`
}

type PipxVenv struct {
	Metadata PipxPackageMetadata `json:"metadata"`
}

type PipxListOutput struct {
	Venvs map[string]PipxVenv `json:"venvs"`
}

type NpmDependency struct {
	Version string `json:"version"`
}

type NpmListOutput struct {
	Dependencies map[string]NpmDependency `json:"dependencies"`
}

func compareUserPackages(user, manager string, desiredPackages []string, runner system.CommandRunner) []actions.Action {
	var a []actions.Action

	// Discover current state
	command := manager + " list --json"
	out, err := runner.Run(user, command)
	if err != nil {
		// Handle case where user or manager is not found, or command fails
		fmt.Printf("Warning: could not list %s packages for user %s: %v\n", manager, user, err)
		return a
	}

	installedPackages := []string{}

	switch manager {
	case "pipx":
		var pipxOutput PipxListOutput
		if err := json.Unmarshal(out, &pipxOutput); err != nil {
			fmt.Printf("Warning: could not parse pipx list output for user %s: %v\n", user, err)
			return a
		}
		for _, venv := range pipxOutput.Venvs {
			installedPackages = append(installedPackages, venv.Metadata.Package)
		}
	case "npm":
		var npmOutput NpmListOutput
		if err := json.Unmarshal(out, &npmOutput); err != nil {
			fmt.Printf("Warning: could not parse npm list output for user %s: %v\n", user, err)
			return a
		}
		for pkg := range npmOutput.Dependencies {
			installedPackages = append(installedPackages, pkg)
		}
	}

	currentMap := make(map[string]bool)
	for _, p := range installedPackages {
		currentMap[p] = true
	}

	desiredMap := make(map[string]bool)
	for _, p := range desiredPackages {
		desiredMap[p] = true
	}

	for pkg := range desiredMap {
		if !currentMap[pkg] {
			a = append(a, &actions.UserPackageAction{User: user, Manager: manager, Package: pkg, State: model.PackageStatePresent})
		}
	}

	for pkg := range currentMap {
		if !desiredMap[pkg] {
			a = append(a, &actions.UserPackageAction{User: user, Manager: manager, Package: pkg, State: model.PackageStateAbsent})
		}
	}

	return a
}

func calculatePackageActions(desired []model.PackageState, current []model.PackageState) []actions.Action {
	var a []actions.Action

	desiredMap := make(map[string]model.PackageState)
	for _, p := range desired {
		desiredMap[p.Name] = p
	}

	currentMap := make(map[string]model.PackageState)
	for _, p := range current {
		currentMap[p.Name] = p
	}

	for name := range desiredMap {
		if _, ok := currentMap[name]; !ok {
			a = append(a, &actions.PackageInstallAction{PackageName: name})
		}
	}

	for name := range currentMap {
		if _, ok := desiredMap[name]; !ok {
			a = append(a, &actions.PackageRemoveAction{PackageName: name})
		}
	}

	return a
}

func calculateServiceActions(desired []model.ServiceState, current []model.ServiceState) []actions.Action {
	var a []actions.Action

	desiredMap := make(map[string]model.ServiceState)
	for _, s := range desired {
		desiredMap[s.Name] = s
	}

	currentMap := make(map[string]model.ServiceState)
	for _, s := range current {
		currentMap[s.Name] = s
	}

	for name, desiredService := range desiredMap {
		if currentService, ok := currentMap[name]; ok {
			if desiredService.Enabled && !currentService.Enabled {
				a = append(a, &actions.ServiceEnableAction{ServiceName: name, Runlevel: desiredService.Runlevel})
			} else if !desiredService.Enabled && currentService.Enabled {
				a = append(a, &actions.ServiceDisableAction{ServiceName: name, Runlevel: currentService.Runlevel})
			}
		} else {
			if desiredService.Enabled {
				a = append(a, &actions.ServiceEnableAction{ServiceName: name, Runlevel: desiredService.Runlevel})
			}
		}
	}

	for name, currentService := range currentMap {
		if _, ok := desiredMap[name]; !ok {
			if currentService.Enabled {
				a = append(a, &actions.ServiceDisableAction{ServiceName: name, Runlevel: currentService.Runlevel})
			}
		}
	}

	return a
}

func calculateUserActions(desired []model.UserState, current []model.UserState, runner system.CommandRunner) ([]actions.Action, error) {
	plan := []actions.Action{}

	// Infer current system groups
	currentSystemGroups, err := inferCurrentSystemGroups(runner)
	if err != nil {
		return nil, fmt.Errorf("failed to infer current system groups: %w", err)
	}

	// Collect all required groups from desired users
	requiredGroups := make(map[string]struct{})
	for _, user := range desired {
		for _, groupName := range user.Groups {
			requiredGroups[groupName] = struct{}{}
		}
	}

	// Create missing groups
	for groupName := range requiredGroups {
		if _, exists := currentSystemGroups[groupName]; !exists {
			plan = append(plan, &actions.GroupCreateAction{GroupName: groupName})
			// Add to currentSystemGroups to prevent duplicate actions
			currentSystemGroups[groupName] = struct{}{}
		}
	}

	// Create maps for users
	currentUsersMap := make(map[string]model.UserState)
	for _, u := range current {
		currentUsersMap[u.Name] = u
	}

	for _, desiredUser := range desired {
		currentUser, userExists := currentUsersMap[desiredUser.Name]

		if !userExists {
			// Create new user and add to groups
			plan = append(plan, &actions.UserCreateAction{UserName: desiredUser.Name})
			for _, groupName := range desiredUser.Groups {
				plan = append(plan, &actions.AddUserToGroupAction{UserName: desiredUser.Name, GroupName: groupName})
			}
		} else {
			// Update existing user's groups
			desiredGroups := make(map[string]struct{})
			for _, g := range desiredUser.Groups {
				desiredGroups[g] = struct{}{}
			}

			currentGroups := make(map[string]struct{})
			for _, g := range currentUser.Groups {
				currentGroups[g] = struct{}{}
			}

			// Add to new groups
			for groupName := range desiredGroups {
				if _, exists := currentGroups[groupName]; !exists {
					plan = append(plan, &actions.AddUserToGroupAction{UserName: desiredUser.Name, GroupName: groupName})
				}
			}

			// Remove from unwanted groups
			for groupName := range currentGroups {
				if _, exists := desiredGroups[groupName]; !exists {
					// Skip removing from primary group
					if groupName == currentUser.PrimaryGroup {
						continue
					}
					plan = append(plan, &actions.RemoveUserFromGroupAction{UserName: desiredUser.Name, GroupName: groupName})
				}
			}
		}
	}

	// Handle user removal (existing logic)
	desiredMap := make(map[string]model.UserState)
	for _, u := range desired {
		desiredMap[u.Name] = u
	}

	for name := range currentUsersMap {
		if _, ok := desiredMap[name]; !ok {
			plan = append(plan, &actions.UserRemoveAction{UserName: name})
		}
	}

	return plan, nil
}

// inferCurrentSystemGroups retrieves the list of current system groups
func inferCurrentSystemGroups(runner system.CommandRunner) (map[string]struct{}, error) {
	output, err := runner.Run("", "sh -c 'cat "+groupFilePath+"'")
	if err != nil {
		return nil, fmt.Errorf("failed to get current system groups: %w", err)
	}

	currentSystemGroups := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 && parts[0] != "" {
				currentSystemGroups[parts[0]] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading system groups output: %w", err)
	}
	return currentSystemGroups, nil
}

func calculateConfigActions(desired *model.SystemState, current *model.SystemState, pruneUnmanaged bool) []actions.Action {
	var a []actions.Action

	// Helper function to check if a path should be ignored.
	// Implemented as closure to access desired.IgnoredConfigs from parent scope.
	isIgnored := func(path string) bool {
		for _, pattern := range desired.IgnoredConfigs {
			if MatchesGlob(pattern, path) {
				return true
			}
		}
		return false
	}

	desiredMap := make(map[string]model.SystemConfigState)
	for _, c := range desired.Configs {
		if !isIgnored(c.Path) {
			desiredMap[c.Path] = c
		}
	}

	currentMap := make(map[string]model.SystemConfigState)
	for _, c := range current.Configs {
		if !isIgnored(c.Path) {
			currentMap[c.Path] = c
		}
	}

	for path, desiredConfig := range desiredMap {
		if currentConfig, ok := currentMap[path]; ok {
			if desiredConfig.Content != currentConfig.Content {
				a = append(a, &actions.FileUpdateAction{Path: path, NewContent: desiredConfig.Content})
			}
			if desiredConfig.Mode != "" && desiredConfig.Mode != currentConfig.Mode {
				a = append(a, &actions.FileChmodAction{Path: path, Mode: desiredConfig.Mode})
			}
			if (desiredConfig.Owner != "" && desiredConfig.Owner != currentConfig.Owner) || (desiredConfig.Group != "" && desiredConfig.Group != currentConfig.Group) {
				a = append(a, &actions.FileChownAction{Path: path, Owner: desiredConfig.Owner, Group: desiredConfig.Group})
			}
		} else {
			a = append(a, &actions.FileCreateAction{Path: path, Content: desiredConfig.Content, Mode: desiredConfig.Mode, Owner: desiredConfig.Owner, Group: desiredConfig.Group})
		}
	}

	for path, currentConfig := range currentMap {
		if _, ok := desiredMap[path]; !ok {
			switch currentConfig.Origin {
			case model.OriginUserCreated:
				if pruneUnmanaged {
					a = append(a, &actions.FileDeleteAction{Path: path})
				} else if !isIgnored(path) {
					fmt.Fprintf(os.Stderr, unmanagedFileWarning, path)
				}
			case model.OriginPackageModified:
				a = append(a, &actions.FileRevertAction{Path: path, OwnerPackage: currentConfig.OriginPackage})
			}
		}
	}

	return a
}
