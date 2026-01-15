//go:build integration
// +build integration

package integration

import (
	"bytes"
	"log/slog"
	"strings"
	"summit/pkg/actions"
	"summit/pkg/config"
	"summit/pkg/diff"
	"summit/pkg/log"
	"summit/pkg/system"
	"testing"
)

func TestBasicConfigLoadAndInfer(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Test loading config and inferring state
	configPath := "/app/test/integration/testdata/create_file.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	t.Logf("IgnoredConfigs: %v", desired.IgnoredConfigs)

	runner := &system.LiveCommandRunner{}
	current, _, err := system.InferSystemState(runner, false)
	if err != nil {
		t.Fatalf("Failed to infer system state: %v", err)
	}

	plan, err := diff.CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Failed to calculate plan: %v", err)
	}

	if len(plan) == 0 {
		t.Log("No actions needed (system already matches desired state)")
		return
	}

	// Log the plan
	t.Logf("Plan has %d actions", len(plan))
	for _, action := range plan {
		t.Logf("Action: %s", action.Description())
	}
}

func TestIncludesIntegration(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Test loading desktop host config with includes
	configPath := "/app/test/integration/testdata/includes/hosts/test-desktop.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to load config with includes: %v", err)
	}

	// Verify merged packages (base + desktop + host)
	expectedPackages := []string{"htop", "git", "openssh", "sway", "foot", "firefox", "chromium"}
	actualPackages := make([]string, len(desired.Packages))
	for i, pkg := range desired.Packages {
		actualPackages[i] = pkg.Name
	}
	for _, expected := range expectedPackages {
		found := false
		for _, actual := range actualPackages {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected package %s not found in merged config", expected)
		}
	}

	// Verify merged services (base overridden by host)
	expectedServices := map[string]bool{
		"sshd":    false, // overridden
		"elogind": true,
	}
	for _, svc := range desired.Services {
		if expected, exists := expectedServices[svc.Name]; exists {
			if svc.Enabled != expected {
				t.Errorf("Service %s enabled=%v, expected %v", svc.Name, svc.Enabled, expected)
			}
		}
	}

	// Verify merged users (base + host groups)
	expectedGroups := []string{"users", "video", "audio", "wheel"}
	for _, user := range desired.Users {
		if user.Name == "testuser" {
			for _, expected := range expectedGroups {
				found := false
				for _, actual := range user.Groups {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected group %s not found in user %s", expected, user.Name)
				}
			}
		}
	}

	t.Logf("Successfully loaded and merged config with %d packages, %d services, %d users",
		len(desired.Packages), len(desired.Services), len(desired.Users))
}

func TestIncludesAdvancedMerging(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Test loading config that merges IgnoredConfigs and UserPackages
	configPath := "/app/test/integration/testdata/includes/hosts/test-advanced.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to load advanced config with includes: %v", err)
	}

	// Verify IgnoredConfigs are merged (union of base + host)
	expectedIgnored := []string{
		"/etc/conf.d/*.conf",
		"/etc/local.d/*.start",
		"/etc/network/interfaces",
		"/etc/resolv.conf",
	}
	actualIgnored := desired.IgnoredConfigs
	for _, expected := range expectedIgnored {
		found := false
		for _, actual := range actualIgnored {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected ignored config %s not found in merged config", expected)
		}
	}

	// Verify UserPackages are merged (union within each manager)
	for _, up := range desired.UserPackages {
		if up.User == "devuser" {
			// Pipx should have union of base + host
			expectedPipx := []string{"black", "flake8", "pytest", "mypy"}
			for _, expected := range expectedPipx {
				found := false
				for _, actual := range up.Pipx {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected pipx package %s not found for user devuser", expected)
				}
			}

			// Npm should have union of base + host
			expectedNpm := []string{"eslint", "prettier", "typescript"}
			for _, expected := range expectedNpm {
				found := false
				for _, actual := range up.Npm {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected npm package %s not found for user devuser", expected)
				}
			}
		}
	}

	t.Logf("Successfully verified IgnoredConfigs and UserPackages merging")
}

func TestCircularIncludesDetection(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Test that circular includes are properly detected and rejected
	configPath := "/app/test/integration/testdata/includes/circular/a.yaml"
	_, err := config.LoadConfig(configPath, logger)
	if err == nil {
		t.Fatal("Expected error for circular includes, but got none")
	}

	// Verify error message mentions circular include
	if !strings.Contains(err.Error(), "circular include") {
		t.Errorf("Error should mention 'circular include', got: %v", err)
	}

	t.Logf("Successfully detected circular include: %v", err)
}

func TestApplyWithRollback(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	// Test applying changes with potential rollback
	configPath := "/app/test/integration/testdata/simple.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	var logBuf bytes.Buffer
	logger = log.NewSlogLogger(slog.LevelDebug, &logBuf)

	runner := &system.LiveCommandRunner{}
	current, _, err := system.InferSystemState(runner, false)
	if err != nil {
		t.Fatalf("Failed to infer system state: %v", err)
	}

	plan, err := diff.CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Failed to calculate plan: %v", err)
	}

	// Filter plan to only include safe actions (installs and creates, no removes)
	var safePlan []actions.Action
	for _, action := range plan {
		desc := action.Description()
		if strings.Contains(desc, "Install package") || strings.Contains(desc, "Create file") {
			safePlan = append(safePlan, action)
		}
	}

	if len(safePlan) == 0 {
		t.Log("No safe actions to apply")
		return
	}

	completedActions := []actions.Action{}
	failed := false

	for _, action := range safePlan {
		t.Logf("Applying: %s", action.Description())
		if err := action.Apply(runner, logger); err != nil {
			t.Logf("Apply failed: %v, rolling back", err)
			failed = true
			break
		}
		completedActions = append(completedActions, action)
	}

	if failed {
		// Rollback
		for i := len(completedActions) - 1; i >= 0; i-- {
			action := completedActions[i]
			t.Logf("Rolling back: %s", action.Description())
			if err := action.Rollback(runner, logger); err != nil {
				t.Errorf("Rollback failed: %v", err)
			}
		}
		t.Fatal("Apply failed and rolled back")
	}

	t.Log("Apply completed successfully")
}
