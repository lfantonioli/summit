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

func TestUserGroupManagement(t *testing.T) {
	logger := log.NewSlogLogger(slog.LevelInfo, &bytes.Buffer{})

	configPath := "/app/test/integration/testdata/user_groups.yaml"
	desired, err := config.LoadConfig(configPath, logger)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	runner := &system.LiveCommandRunner{}
	current, _, err := system.InferSystemState(runner, false)
	if err != nil {
		t.Fatalf("Failed to infer system state: %v", err)
	}

	plan, err := diff.CalculatePlan(desired, current, runner, false)
	if err != nil {
		t.Fatalf("Failed to calculate plan: %v", err)
	}

	// Assert that the plan includes expected actions
	hasGroupCreate := false
	hasUserCreate := false
	hasAddToGroup := false

	for _, action := range plan {
		desc := action.Description()
		if strings.Contains(desc, "Create group") {
			hasGroupCreate = true
		}
		if strings.Contains(desc, "Create user") {
			hasUserCreate = true
		}
		if strings.Contains(desc, "Add user") && strings.Contains(desc, "to group") {
			hasAddToGroup = true
		}
	}

	if !hasGroupCreate {
		t.Error("Expected at least one group creation action")
	}
	if !hasUserCreate {
		t.Error("Expected user creation actions")
	}
	if !hasAddToGroup {
		t.Error("Expected add user to group actions")
	}

	// Apply the actions to test execution
	var logBuf bytes.Buffer
	logger = log.NewSlogLogger(slog.LevelInfo, &logBuf)

	// Filter to safe actions (creates, no removes)
	var safePlan []actions.Action
	for _, action := range plan {
		desc := action.Description()
		if strings.Contains(desc, "Create group") || strings.Contains(desc, "Create user") || strings.Contains(desc, "Add user") {
			safePlan = append(safePlan, action)
		}
	}

	if len(safePlan) == 0 {
		t.Log("No safe actions to apply")
		return
	}

	completedActions := []actions.Action{}

	for _, action := range safePlan {
		t.Logf("Applying: %s", action.Description())
		if err := action.Apply(runner, logger); err != nil {
			t.Errorf("Apply failed for %s: %v", action.Description(), err)
			// Attempt rollback
			for i := len(completedActions) - 1; i >= 0; i-- {
				rollbackAction := completedActions[i]
				t.Logf("Rolling back: %s", rollbackAction.Description())
				if rbErr := rollbackAction.Rollback(runner, logger); rbErr != nil {
					t.Errorf("Rollback failed for %s: %v", rollbackAction.Description(), rbErr)
				}
			}
			return
		}
		completedActions = append(completedActions, action)
	}

	t.Logf("Successfully applied %d actions", len(completedActions))
}
