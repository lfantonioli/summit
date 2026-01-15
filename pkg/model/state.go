package model

import (
	"fmt"
	"sort"
	"strings"
)

type FileOrigin string

const (
	OriginManaged         FileOrigin = "managed"
	OriginUserCreated     FileOrigin = "user-created"
	OriginPackageModified FileOrigin = "package-modified"
)

type UserPackageActionState string

const (
	PackageStatePresent UserPackageActionState = "present"
	PackageStateAbsent  UserPackageActionState = "absent"
)

// Valid Alpine runlevels
var ValidRunlevels = map[string]bool{
	"boot":      true,
	"default":   true,
	"sysinit":   true,
	"nonetwork": true,
	"shutdown":  true,
}

type ValidationError struct {
	Field   string
	Message string
	Line    int
}

func (e ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s (line %d): %s", e.Field, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, e := range es {
		sb.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
	}
	return sb.String()
}

type Validator interface {
	Validate() ValidationErrors
}

type SystemState struct {
	Includes       []string            `yaml:"includes,omitempty"` // List of config files to include and merge
	Packages       []PackageState      `yaml:"packages"`
	Services       []ServiceState      `yaml:"services"`
	Users          []UserState         `yaml:"users"`
	Configs        []SystemConfigState `yaml:"configs"`
	IgnoredConfigs []string            `yaml:"ignored-configs,omitempty"` // Ignore configs can either be file paths or glob patterns
	UserPackages   []UserPackageState  `yaml:"user-packages,omitempty"`
}

type UserPackageState struct {
	User string   `yaml:"user"`
	Pipx []string `yaml:"pipx,omitempty"`
	Npm  []string `yaml:"npm,omitempty"`
}

type UserState struct {
	Name         string   `yaml:"name"`
	Groups       []string `yaml:"groups"`
	PrimaryGroup string   `yaml:"-"`
}

type PackageState struct {
	Name string `yaml:"name"`
}

type ServiceState struct {
	Name     string `yaml:"name"`
	Enabled  bool   `yaml:"enabled"`
	Runlevel string `yaml:"runlevel"`
}

type SystemConfigState struct {
	Path          string     `yaml:"path"`
	Content       string     `yaml:"content"`
	Mode          string     `yaml:"mode,omitempty"`
	Owner         string     `yaml:"owner,omitempty"`
	Group         string     `yaml:"group,omitempty"`
	Origin        FileOrigin `yaml:"-"` // "managed", "package-modified", "user-created"
	Deleted       bool       `yaml:"-"`
	FileStatus    string     `yaml:"-"`
	OriginPackage string     `yaml:"-"`
}

type IgnoredConfig struct {
	Path   string
	Reason string
}

func (s *SystemState) Sort() {
	// sort packages alphabetically
	sort.Slice(s.Packages, func(i, j int) bool {
		return s.Packages[i].Name < s.Packages[j].Name
	})

	// sort services alphabetically
	sort.Slice(s.Services, func(i, j int) bool {
		return s.Services[i].Name < s.Services[j].Name
	})

	// sort users alphabetically
	sort.Slice(s.Users, func(i, j int) bool {
		return s.Users[i].Name < s.Users[j].Name
	})

	// sort configs alphabetically
	sort.Slice(s.Configs, func(i, j int) bool {
		return s.Configs[i].Path < s.Configs[j].Path
	})

	// sort user packages alphabetically by user
	sort.Slice(s.UserPackages, func(i, j int) bool {
		return s.UserPackages[i].User < s.UserPackages[j].User
	})
}

func (s *SystemState) Validate() ValidationErrors {
	var errs ValidationErrors

	// Validate includes
	for i, include := range s.Includes {
		if strings.TrimSpace(include) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("includes[%d]", i), Message: "include path cannot be empty"})
		}
	}

	// Validate packages
	for i, pkg := range s.Packages {
		if strings.TrimSpace(pkg.Name) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("packages[%d].name", i), Message: "package name cannot be empty"})
		}
		if !isValidPackageName(pkg.Name) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("packages[%d].name", i), Message: "package name contains invalid characters (only alphanumeric, hyphens, and dots allowed)"})
		}
	}

	// Validate services
	for i, svc := range s.Services {
		if strings.TrimSpace(svc.Name) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("services[%d].name", i), Message: "service name cannot be empty"})
		}
		// Empty runlevel is valid for disabled services (not added to any runlevel)
		if svc.Runlevel != "" && !ValidRunlevels[svc.Runlevel] {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("services[%d].runlevel", i), Message: fmt.Sprintf("invalid runlevel '%s', must be one of: boot, default, sysinit, nonetwork, shutdown", svc.Runlevel)})
		}
	}

	// Validate users
	for i, user := range s.Users {
		if strings.TrimSpace(user.Name) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("users[%d].name", i), Message: "user name cannot be empty"})
		}
		if !isValidUserName(user.Name) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("users[%d].name", i), Message: "user name contains invalid characters (only lowercase letters, numbers, hyphens, and underscores allowed)"})
		}
		for j, group := range user.Groups {
			if !isValidUserName(group) {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("users[%d].groups[%d]", i, j), Message: "group name contains invalid characters"})
			}
		}
	}

	// Validate configs
	for i, cfg := range s.Configs {
		if strings.TrimSpace(cfg.Path) == "" {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].path", i), Message: "config path cannot be empty"})
		}
		if !strings.HasPrefix(cfg.Path, "/") {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].path", i), Message: "config path must be absolute (start with '/')"})
		}
		if strings.Contains(cfg.Path, "..") {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].path", i), Message: "config path cannot contain '..'"})
		}
		// Check for conflicts with intrinsic ignores
		if isIntrinsicIgnore(cfg.Path) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].path", i), Message: "cannot manage intrinsically ignored file (security/safety reasons)"})
		}
		if cfg.Mode != "" {
			if !isValidOctalMode(cfg.Mode) {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].mode", i), Message: "mode must be a valid octal value like '0755' or '0644'"})
			}
		}
		if cfg.Owner != "" && !isValidUserName(cfg.Owner) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].owner", i), Message: "owner contains invalid characters"})
		}
		if cfg.Group != "" && !isValidUserName(cfg.Group) {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("configs[%d].group", i), Message: "group contains invalid characters"})
		}
	}

	// Validate user packages
	userMap := make(map[string]bool)
	for _, user := range s.Users {
		userMap[user.Name] = true
	}
	for i, up := range s.UserPackages {
		if !userMap[up.User] {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("user-packages[%d].user", i), Message: fmt.Sprintf("user '%s' not defined in users section", up.User)})
		}
		for j, pkg := range up.Pipx {
			if !isValidPackageName(pkg) {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("user-packages[%d].pipx[%d]", i, j), Message: "package name contains invalid characters"})
			}
		}
		for j, pkg := range up.Npm {
			if !isValidPackageName(pkg) {
				errs = append(errs, ValidationError{Field: fmt.Sprintf("user-packages[%d].npm[%d]", i, j), Message: "package name contains invalid characters"})
			}
		}
	}

	return errs
}

func isValidPackageName(name string) bool {
	for _, r := range name {
		if r < 32 || r == 127 { // control chars
			return false
		}
	}
	return true
}

func isValidUserName(name string) bool {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func isValidOctalMode(mode string) bool {
	if len(mode) != 4 || mode[0] != '0' {
		return false
	}
	for _, r := range mode[1:] {
		if r < '0' || r > '7' {
			return false
		}
	}
	return true
}

func isIntrinsicIgnore(path string) bool {
	// Exact matches
	intrinsicIgnores := []string{
		"/etc/passwd",
		"/etc/group",
		"/etc/shadow",
		"/etc/apk/world",
	}
	for _, ignore := range intrinsicIgnores {
		if path == ignore {
			return true
		}
	}
	// Prefix ignores
	prefixIgnores := []string{
		"/etc/apk/keys",
		"/etc/runlevels",
	}
	for _, ignore := range prefixIgnores {
		if strings.HasPrefix(path, ignore) {
			return true
		}
	}
	// Suffix ignores
	if strings.HasSuffix(path, "-") || strings.HasSuffix(path, ".bak") {
		return true
	}
	return false
}
