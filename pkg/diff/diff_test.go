package diff

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"summit/pkg/actions"
	"summit/pkg/model"
	"testing"
)

// MockCommandRunner is a mock implementation of the CommandRunner for testing.
type MockCommandRunner struct {
	Responses map[string][]byte
	Errors    map[string]error
}

// Run simulates running a command.
func (r *MockCommandRunner) Run(user, command string) ([]byte, error) {
	key := fmt.Sprintf("%s:%s", user, command)
	if err, ok := r.Errors[key]; ok {
		return nil, err
	}
	if resp, ok := r.Responses[key]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for %s", key)
}

func TestCalculatePlanWithUserPackages(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
		},
		Users: []model.UserState{
			{Name: "mino"},
		},
		UserPackages: []model.UserPackageState{
			{
				User: "mino",
				Pipx: []string{"black", "ruff"},
			},
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "pipx"},
		},
		Users: []model.UserState{
			{Name: "mino"},
		},
	}

	// Create a mock runner
	runner := &MockCommandRunner{
		Responses: map[string][]byte{
			":sh -c 'cat /etc/group'": []byte(""),
			"mino:pipx list --json": []byte(`{
		                "venvs": {
		                    "black": {
		                        "metadata": {
		                            "package": "black"
		                        }
		                    },
		                    "poetry": {
		                        "metadata": {
		                            "package": "poetry"
		                        }
		                    }
		                }
		            }`),
			"mino:npm list --json": []byte(`{}`),
		},
		Errors: make(map[string]error),
	}
	// For this test, we will call the function directly.
	plan, err := CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Error calculating plan: %v", err)
	}

	expected := []actions.Action{
		&actions.UserPackageAction{User: "mino", Manager: "pipx", Package: "ruff", State: model.PackageStatePresent},
		&actions.UserPackageAction{User: "mino", Manager: "pipx", Package: "poetry", State: model.PackageStateAbsent},
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Description() < plan[j].Description()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Description() < expected[j].Description()
	})

	if !reflect.DeepEqual(plan, expected) {
		t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, expected)
	}
}

func TestCalculatePlanWithUserPackagesDependencyFailure(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{},
		UserPackages: []model.UserPackageState{
			{
				User: "mino",
				Pipx: []string{"black", "ruff"},
			},
		},
	}

	current := &model.SystemState{}

	// Create a mock runner
	runner := &MockCommandRunner{}

	// We expect a validation error because the 'pipx' package and the 'mino' user are missing.
	_, err := CalculatePlan(desired, current, runner, false)
	if err == nil {
		t.Fatal("Expected a validation error, but got nil")
	}

	expectedError := "dependency validation failed:\n  - user packages require 'pipx' to be installed for packages: black, ruff. Add 'pipx' to the system packages list.\n  - user 'mino' not found for user-packages"
	if err.Error() != expectedError {
		t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedError, err.Error())
	}
}

func TestCalculatePlanWithIgnoredConfigs(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "package1"},
		},
		Configs: []model.SystemConfigState{
			{Path: "/etc/my-app/config.json", Content: "new content"},
			{Path: "/etc/should-be-created.conf", Content: "created"},
		},
		IgnoredConfigs: []string{
			"/etc/my-app/config.json", // Ignore exact path
			"/var/log/*.log",          // Ignore with glob
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{},
		Configs: []model.SystemConfigState{
			{Path: "/etc/my-app/config.json", Content: "old content", Origin: "user-created"},
			{Path: "/var/log/app.log", Content: "log data", Origin: "user-created"},
			{Path: "/var/log/another.log", Content: "log data", Origin: "user-created"},
		},
	}

	expected := []actions.Action{
		&actions.PackageInstallAction{PackageName: "package1"},
		&actions.FileCreateAction{Path: "/etc/should-be-created.conf", Content: "created"},
	}

	// Create a mock runner
	runner := &MockCommandRunner{
		Responses: map[string][]byte{":sh -c 'cat /etc/group'": []byte("")},
	}

	plan, err := CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Error calculating plan: %v", err)
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Description() < plan[j].Description()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Description() < expected[j].Description()
	})

	if !reflect.DeepEqual(plan, expected) {
		t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, expected)
	}
}

func TestCalculatePlanSuppressesWarningsForIgnoredUnmanagedFiles(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "package1"},
		},
		Configs: []model.SystemConfigState{
			{Path: "/etc/managed.conf", Content: "managed"},
		},
		IgnoredConfigs: []string{
			"/etc/ignored.conf", // Exact match
			"/var/log/*.log",    // Glob pattern
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{},
		Configs: []model.SystemConfigState{
			{Path: "/etc/managed.conf", Content: "old", Origin: "user-created"},
			// Unmanaged files - not in desired configs
			{Path: "/etc/ignored.conf", Content: "ignored content", Origin: "user-created"},
			{Path: "/var/log/app.log", Content: "log", Origin: "user-created"},
			{Path: "/etc/unignored.conf", Content: "unignored content", Origin: "user-created"},
		},
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	runner := &MockCommandRunner{
		Responses: map[string][]byte{":sh -c 'cat /etc/group'": []byte("")},
	}

	plan, err := CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Error calculating plan: %v", err)
	}

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = oldStderr

	stderrOutput := buf.String()

	// Verify plan contains expected actions
	expected := []actions.Action{
		&actions.PackageInstallAction{PackageName: "package1"},
		&actions.FileUpdateAction{Path: "/etc/managed.conf", NewContent: "managed"},
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Description() < plan[j].Description()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Description() < expected[j].Description()
	})

	if !reflect.DeepEqual(plan, expected) {
		t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, expected)
	}

	// Verify warnings: should warn for /etc/unignored.conf but not for ignored files
	if !strings.Contains(stderrOutput, "Warning: unmanaged file found /etc/unignored.conf") {
		t.Errorf("Expected warning for unignored unmanaged file /etc/unignored.conf, but stderr was: %s", stderrOutput)
	}

	if strings.Contains(stderrOutput, "Warning: unmanaged file found /etc/ignored.conf") {
		t.Errorf("Unexpected warning for ignored file /etc/ignored.conf, stderr: %s", stderrOutput)
	}

	if strings.Contains(stderrOutput, "Warning: unmanaged file found /var/log/app.log") {
		t.Errorf("Unexpected warning for ignored file /var/log/app.log, stderr: %s", stderrOutput)
	}
}

func TestCalculateUserActions_GroupManagement(t *testing.T) {
	tests := []struct {
		name     string
		desired  []model.UserState
		current  []model.UserState
		mockResp map[string][]byte
		expected []actions.Action
	}{
		{
			name: "New user with groups requiring group creation",
			desired: []model.UserState{
				{Name: "newuser", Groups: []string{"wheel", "newgroup"}},
			},
			current: []model.UserState{},
			mockResp: map[string][]byte{
				":sh -c 'cat /etc/group'": []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nadm:x:4:\nwheel:x:10:\n"),
			},
			expected: []actions.Action{
				&actions.GroupCreateAction{GroupName: "newgroup"},
				&actions.UserCreateAction{UserName: "newuser"},
				&actions.AddUserToGroupAction{UserName: "newuser", GroupName: "wheel"},
				&actions.AddUserToGroupAction{UserName: "newuser", GroupName: "newgroup"},
			},
		},
		{
			name: "Existing user add to new groups",
			desired: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel", "newgroup"}},
			},
			current: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel"}},
			},
			mockResp: map[string][]byte{
				":sh -c 'cat /etc/group'": []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nadm:x:4:\nwheel:x:10:\n"),
			},
			expected: []actions.Action{
				&actions.GroupCreateAction{GroupName: "newgroup"},
				&actions.AddUserToGroupAction{UserName: "existinguser", GroupName: "newgroup"},
			},
		},
		{
			name: "Existing user remove from groups",
			desired: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel"}},
			},
			current: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel", "oldgroup"}},
			},
			mockResp: map[string][]byte{
				":sh -c 'cat /etc/group'": []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nadm:x:4:\nwheel:x:10:\noldgroup:x:1000:\n"),
			},
			expected: []actions.Action{
				&actions.RemoveUserFromGroupAction{UserName: "existinguser", GroupName: "oldgroup"},
			},
		},
		{
			name: "Mixed operations",
			desired: []model.UserState{
				{Name: "newuser", Groups: []string{"wheel"}},
				{Name: "existinguser", Groups: []string{"newgroup"}},
			},
			current: []model.UserState{
				{Name: "existinguser", Groups: []string{"oldgroup"}},
			},
			mockResp: map[string][]byte{
				":sh -c 'cat /etc/group'": []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nadm:x:4:\nwheel:x:10:\noldgroup:x:1000:\n"),
			},
			expected: []actions.Action{
				&actions.GroupCreateAction{GroupName: "newgroup"},
				&actions.UserCreateAction{UserName: "newuser"},
				&actions.AddUserToGroupAction{UserName: "newuser", GroupName: "wheel"},
				&actions.AddUserToGroupAction{UserName: "existinguser", GroupName: "newgroup"},
				&actions.RemoveUserFromGroupAction{UserName: "existinguser", GroupName: "oldgroup"},
			},
		},
		{
			name: "No changes",
			desired: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel"}},
			},
			current: []model.UserState{
				{Name: "existinguser", Groups: []string{"wheel"}},
			},
			mockResp: map[string][]byte{
				":sh -c 'cat /etc/group'": []byte("root:x:0:\nbin:x:1:\ndaemon:x:2:\nsys:x:3:\nadm:x:4:\nwheel:x:10:\n"),
			},
			expected: []actions.Action{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &MockCommandRunner{
				Responses: tt.mockResp,
				Errors:    make(map[string]error),
			}

			plan, err := calculateUserActions(tt.desired, tt.current, runner)
			if err != nil {
				t.Fatalf("calculateUserActions failed: %v", err)
			}

			// Sort both slices for comparison
			sort.Slice(plan, func(i, j int) bool {
				return plan[i].Description() < plan[j].Description()
			})
			sort.Slice(tt.expected, func(i, j int) bool {
				return tt.expected[i].Description() < tt.expected[j].Description()
			})

			if !reflect.DeepEqual(plan, tt.expected) {
				t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, tt.expected)
			}
		})
	}
}

func TestCalculatePlanUnmanagedFilesDefaultBehavior(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "package1"},
		},
		Configs: []model.SystemConfigState{
			{Path: "/etc/managed-config.conf", Content: "managed content"},
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{},
		Configs: []model.SystemConfigState{
			{Path: "/etc/unmanaged-file.conf", Content: "unmanaged content", Origin: model.OriginUserCreated},
			{Path: "/etc/modified-package-file.conf", Content: "modified content", Origin: model.OriginPackageModified, OriginPackage: "somepackage"},
		},
	}

	runner := &MockCommandRunner{
		Responses: map[string][]byte{":sh -c 'cat /etc/group'": []byte("")},
	}

	// Test default behavior (pruneUnmanaged = false)
	plan, err := CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Error calculating plan: %v", err)
	}

	// Should include package install, file create, and file revert, but NO file delete
	expected := []actions.Action{
		&actions.PackageInstallAction{PackageName: "package1"},
		&actions.FileCreateAction{Path: "/etc/managed-config.conf", Content: "managed content"},
		&actions.FileRevertAction{Path: "/etc/modified-package-file.conf", OwnerPackage: "somepackage"},
	}

	// Ensure no FileDeleteAction is present
	for _, action := range plan {
		if _, ok := action.(*actions.FileDeleteAction); ok {
			t.Errorf("Expected no FileDeleteAction in default behavior, but found: %s", action.Description())
		}
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Description() < plan[j].Description()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Description() < expected[j].Description()
	})

	if !reflect.DeepEqual(plan, expected) {
		t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, expected)
	}
}

func TestCalculatePlanUnmanagedFilesPruneEnabled(t *testing.T) {
	desired := &model.SystemState{
		Packages: []model.PackageState{
			{Name: "package1"},
		},
		Configs: []model.SystemConfigState{
			{Path: "/etc/managed-config.conf", Content: "managed content"},
		},
	}

	current := &model.SystemState{
		Packages: []model.PackageState{},
		Configs: []model.SystemConfigState{
			{Path: "/etc/unmanaged-file.conf", Content: "unmanaged content", Origin: model.OriginUserCreated},
			{Path: "/etc/modified-package-file.conf", Content: "modified content", Origin: model.OriginPackageModified, OriginPackage: "somepackage"},
		},
	}

	runner := &MockCommandRunner{
		Responses: map[string][]byte{":sh -c 'cat /etc/group'": []byte("")},
	}

	// Test with pruneUnmanaged = true
	plan, err := CalculatePlan(desired, current, runner, true)
	if err != nil {
		t.Fatalf("Error calculating plan: %v", err)
	}

	// Should include package install, file create, file revert, AND file delete
	expected := []actions.Action{
		&actions.PackageInstallAction{PackageName: "package1"},
		&actions.FileCreateAction{Path: "/etc/managed-config.conf", Content: "managed content"},
		&actions.FileRevertAction{Path: "/etc/modified-package-file.conf", OwnerPackage: "somepackage"},
		&actions.FileDeleteAction{Path: "/etc/unmanaged-file.conf"},
	}

	// Ensure exactly one FileDeleteAction is present
	deleteActions := 0
	for _, action := range plan {
		if _, ok := action.(*actions.FileDeleteAction); ok {
			deleteActions++
		}
	}
	if deleteActions != 1 {
		t.Errorf("Expected exactly 1 FileDeleteAction when pruning enabled, but found: %d", deleteActions)
	}

	sort.Slice(plan, func(i, j int) bool {
		return plan[i].Description() < plan[j].Description()
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Description() < expected[j].Description()
	})

	if !reflect.DeepEqual(plan, expected) {
		t.Errorf("Plan not as expected:\nGot:      %+v\nExpected: %+v", plan, expected)
	}
}

func TestMatchesGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"/etc/*.txt", "/etc/file.txt", true},
		{"/etc/*.txt", "/etc/file.conf", false},
		{"/etc/ssh/**", "/etc/ssh/sshd_config", true},
		{"/etc/ssh/**", "/etc/ssh/subdir/file", true},
		{"/etc/ssh/**", "/etc/other/file", false},
		{"/etc/ssh/**/*.pub", "/etc/ssh/host.pub", true},
		{"/etc/ssh/**/*.pub", "/etc/ssh/keys/host.pub", true},
		{"/etc/ssh/**/*.pub", "/etc/ssh/host.conf", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.pattern, tt.path), func(t *testing.T) {
			result := MatchesGlob(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("MatchesGlob(%q, %q) = %v; want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}
