package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"summit/pkg/log"
	"summit/pkg/model"
	"summit/pkg/system"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

func LoadConfig(filename string, logger log.Logger) (*model.SystemState, error) {
	cfg, err := loadConfigFile(filename, logger)
	if err != nil {
		return nil, err
	}

	// Validate includes before processing
	if errs := validateIncludes(cfg.Includes); len(errs) > 0 {
		return nil, errs
	}

	// Process includes recursively
	if len(cfg.Includes) > 0 {
		cfg, err = processIncludes(cfg, filename, logger)
		if err != nil {
			return nil, err
		}
	}

	if errs := cfg.Validate(); len(errs) > 0 {
		return nil, errs
	}

	cfg.Sort()

	return &cfg, nil
}

// processIncludes processes the includes field of a SystemState, loading and merging
// included configuration files recursively.
func processIncludes(cfg model.SystemState, baseFile string, logger log.Logger) (model.SystemState, error) {
	visited := make(map[string]bool) // For cycle detection
	return processIncludesRecursive(cfg, baseFile, visited, logger)
}

func processIncludesRecursive(cfg model.SystemState, baseFile string, visited map[string]bool, logger log.Logger) (model.SystemState, error) {
	result := &model.SystemState{}

	// Track this file to prevent cycles
	absBase, err := filepath.Abs(baseFile)
	if err != nil {
		return model.SystemState{}, fmt.Errorf("failed to resolve absolute path for %s: %w", baseFile, err)
	}
	if visited[absBase] {
		return model.SystemState{}, fmt.Errorf("circular include detected: %s", baseFile)
	}
	visited[absBase] = true

	// Process each include in order
	for _, includePath := range cfg.Includes {
		resolvedPath := resolveIncludePath(baseFile, includePath)

		includedCfg, err := loadConfigFile(resolvedPath, logger)
		if err != nil {
			return model.SystemState{}, fmt.Errorf("failed to load include '%s': %w", includePath, err)
		}

		// Recursively process nested includes
		if len(includedCfg.Includes) > 0 {
			includedCfg, err = processIncludesRecursive(includedCfg, resolvedPath, visited, logger)
			if err != nil {
				return model.SystemState{}, err
			}
		}

		// Merge included config into result
		result = mergeConfigs(result, &includedCfg, logger)
	}

	// Finally merge the current file's content (highest priority)
	result = mergeConfigs(result, &cfg, logger)

	return *result, nil
}

func loadConfigFile(filename string, logger log.Logger) (model.SystemState, error) {
	f, err := afero.ReadFile(system.AppFs, filename)
	if err != nil {
		return model.SystemState{}, err
	}

	var cfg model.SystemState
	err = yaml.Unmarshal(f, &cfg)
	if err != nil {
		return model.SystemState{}, err
	}

	for i := range cfg.Configs {
		cfg.Configs[i].Origin = model.OriginManaged
	}

	return cfg, nil
}

func resolveIncludePath(baseFile, includePath string) string {
	// If absolute path, use as-is
	if filepath.IsAbs(includePath) {
		return includePath
	}

	// Relative to the directory containing baseFile
	baseDir := filepath.Dir(baseFile)
	return filepath.Join(baseDir, includePath)
}

// mergeConfigs merges two SystemState configurations using entity-specific strategies:
// - Packages: union by name
// - Services: last-wins by (name + runlevel) with warnings
// - Users: last-wins for properties, union for groups
// - Configs: last-wins by path
// - UserPackages: union packages within each manager
// - IgnoredConfigs: union all patterns
// The override configuration takes priority over the base.
func mergeConfigs(base, override *model.SystemState, logger log.Logger) *model.SystemState {
	result := &model.SystemState{}

	// Packages: Union by name
	result.Packages = mergePackages(base.Packages, override.Packages)

	// Services: Last-wins by (name + runlevel)
	result.Services = mergeServices(base.Services, override.Services, logger)

	// Users: Last-wins by name, union groups
	result.Users = mergeUsers(base.Users, override.Users, logger)

	// Configs: Last-wins by path
	result.Configs = mergeSystemConfigs(base.Configs, override.Configs, logger)

	// UserPackages: Merge by user, union package lists
	result.UserPackages = mergeUserPackages(base.UserPackages, override.UserPackages, logger)

	// IgnoredConfigs: Union (append all patterns)
	result.IgnoredConfigs = mergeIgnoredConfigs(base.IgnoredConfigs, override.IgnoredConfigs)

	// Note: Includes are NOT merged (already processed)

	return result
}

func mergePackages(base, override []model.PackageState) []model.PackageState {
	seen := make(map[string]bool)
	result := []model.PackageState{}

	for _, pkg := range base {
		result = append(result, pkg)
		seen[pkg.Name] = true
	}

	for _, pkg := range override {
		if !seen[pkg.Name] {
			result = append(result, pkg)
			seen[pkg.Name] = true
		}
	}

	return result
}

func mergeServices(base, override []model.ServiceState, logger log.Logger) []model.ServiceState {
	serviceMap := make(map[string]model.ServiceState)

	// Add base services
	for _, svc := range base {
		key := fmt.Sprintf("%s:%s", svc.Name, svc.Runlevel)
		serviceMap[key] = svc
	}

	// Override with new services (warn on conflicts)
	for _, svc := range override {
		key := fmt.Sprintf("%s:%s", svc.Name, svc.Runlevel)
		if existing, exists := serviceMap[key]; exists {
			// Log warning about override
			logger.Warn("Service overridden",
				"service", svc.Name,
				"runlevel", svc.Runlevel,
				"was_enabled", existing.Enabled,
				"now_enabled", svc.Enabled)
		}
		serviceMap[key] = svc
	}

	// Convert back to slice
	result := []model.ServiceState{}
	for _, svc := range serviceMap {
		result = append(result, svc)
	}

	// Sort by name for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Runlevel < result[j].Runlevel
		}
		return result[i].Name < result[j].Name
	})

	return result
}

func mergeUsers(base, override []model.UserState, logger log.Logger) []model.UserState {
	userMap := make(map[string]model.UserState)

	// Add base users
	for _, user := range base {
		userMap[user.Name] = user
	}

	// Merge override users
	for _, user := range override {
		if existing, exists := userMap[user.Name]; exists {
			// Union the groups
			groupSet := make(map[string]bool)
			for _, g := range existing.Groups {
				groupSet[g] = true
			}
			for _, g := range user.Groups {
				groupSet[g] = true
			}

			// Convert back to slice
			mergedGroups := []string{}
			for g := range groupSet {
				mergedGroups = append(mergedGroups, g)
			}

			// Intentionally modify user.Groups before storing in the map
			// This merges the groups from both base and override configs
			user.Groups = mergedGroups

			logger.Warn("User groups merged", "user", user.Name)
		}
		userMap[user.Name] = user
	}

	result := []model.UserState{}
	for _, user := range userMap {
		result = append(result, user)
	}

	// Sort by name for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func mergeSystemConfigs(base, override []model.SystemConfigState, logger log.Logger) []model.SystemConfigState {
	configMap := make(map[string]model.SystemConfigState)

	for _, cfg := range base {
		configMap[cfg.Path] = cfg
	}

	for _, cfg := range override {
		if _, exists := configMap[cfg.Path]; exists {
			logger.Warn("Config overridden", "path", cfg.Path)
		}
		configMap[cfg.Path] = cfg
	}

	result := []model.SystemConfigState{}
	for _, cfg := range configMap {
		result = append(result, cfg)
	}

	// Sort by path for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result
}

func mergeUserPackages(base, override []model.UserPackageState, logger log.Logger) []model.UserPackageState {
	userPkgMap := make(map[string]model.UserPackageState)

	for _, up := range base {
		userPkgMap[up.User] = up
	}

	for _, up := range override {
		if existing, exists := userPkgMap[up.User]; exists {
			// Union pipx packages
			pipxSet := make(map[string]bool)
			for _, p := range existing.Pipx {
				pipxSet[p] = true
			}
			for _, p := range up.Pipx {
				pipxSet[p] = true
			}

			// Union npm packages
			npmSet := make(map[string]bool)
			for _, p := range existing.Npm {
				npmSet[p] = true
			}
			for _, p := range up.Npm {
				npmSet[p] = true
			}

			// Convert back to slices
			up.Pipx = mapKeysToSlice(pipxSet)
			up.Npm = mapKeysToSlice(npmSet)

			logger.Warn("User packages merged", "user", up.User)
		}
		userPkgMap[up.User] = up
	}

	result := []model.UserPackageState{}
	for _, up := range userPkgMap {
		result = append(result, up)
	}

	// Sort by user for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].User < result[j].User
	})

	return result
}

func mergeIgnoredConfigs(base, override []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, pattern := range base {
		if !seen[pattern] {
			result = append(result, pattern)
			seen[pattern] = true
		}
	}

	for _, pattern := range override {
		if !seen[pattern] {
			result = append(result, pattern)
			seen[pattern] = true
		}
	}

	return result
}

func mapKeysToSlice(m map[string]bool) []string {
	result := []string{}
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func validateIncludes(includes []string) model.ValidationErrors {
	var errs model.ValidationErrors
	for i, include := range includes {
		if strings.TrimSpace(include) == "" {
			errs = append(errs, model.ValidationError{Field: fmt.Sprintf("includes[%d]", i), Message: "include path cannot be empty"})
		}
	}
	return errs
}
