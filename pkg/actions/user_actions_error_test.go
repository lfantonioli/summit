package actions

import (
	"errors"
	"testing"

	"summit/pkg/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserCreateAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *UserCreateAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty username",
			action: &UserCreateAction{
				UserName: "",
			},
			expectError: true,
		},
		{
			name: "user already exists",
			action: &UserCreateAction{
				UserName: "existinguser",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "adduser -D existinguser", errors.New("adduser: user 'existinguser' already exists"))
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name: "invalid username",
			action: &UserCreateAction{
				UserName: "user with spaces",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "adduser -D user with spaces", errors.New("adduser: invalid username 'user with spaces'"))
			},
			expectError: true,
			errorMsg:    "invalid username",
		},
		{
			name: "username too long",
			action: &UserCreateAction{
				UserName: "very-long-username-that-exceeds-the-maximum-allowed-length-for-user-names-in-linux-systems",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "adduser -D very-long-username-that-exceeds-the-maximum-allowed-length-for-user-names-in-linux-systems", errors.New("adduser: username too long"))
			},
			expectError: true,
		},
		{
			name: "special characters in username",
			action: &UserCreateAction{
				UserName: "user@domain",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "adduser -D user@domain", errors.New("adduser: invalid character in username"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestUserRemoveAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *UserRemoveAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty username",
			action: &UserRemoveAction{
				UserName: "",
			},
			expectError: true,
		},
		{
			name: "user does not exist",
			action: &UserRemoveAction{
				UserName: "nonexistentuser",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "deluser nonexistentuser", errors.New("deluser: user 'nonexistentuser' does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "user is currently logged in",
			action: &UserRemoveAction{
				UserName: "loggedinuser",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "deluser loggedinuser", errors.New("deluser: user 'loggedinuser' is currently logged in"))
			},
			expectError: true,
			errorMsg:    "currently logged in",
		},
		{
			name: "cannot remove root user",
			action: &UserRemoveAction{
				UserName: "root",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "deluser root", errors.New("deluser: cannot remove root user"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestGroupCreateAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *GroupCreateAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty group name",
			action: &GroupCreateAction{
				GroupName: "",
			},
			expectError: true,
		},
		{
			name: "group already exists",
			action: &GroupCreateAction{
				GroupName: "existinggroup",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "addgroup existinggroup", errors.New("addgroup: group 'existinggroup' already exists"))
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name: "invalid group name",
			action: &GroupCreateAction{
				GroupName: "group with spaces",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "addgroup group with spaces", errors.New("addgroup: invalid group name"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestAddUserToGroupAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *AddUserToGroupAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty username",
			action: &AddUserToGroupAction{
				UserName:  "",
				GroupName: "wheel",
			},
			expectError: true,
		},
		{
			name: "empty group name",
			action: &AddUserToGroupAction{
				UserName:  "testuser",
				GroupName: "",
			},
			expectError: true,
		},
		{
			name: "user does not exist",
			action: &AddUserToGroupAction{
				UserName:  "nonexistentuser",
				GroupName: "wheel",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "addgroup nonexistentuser wheel", errors.New("addgroup: user 'nonexistentuser' does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "group does not exist",
			action: &AddUserToGroupAction{
				UserName:  "testuser",
				GroupName: "nonexistentgroup",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "addgroup testuser nonexistentgroup", errors.New("addgroup: group 'nonexistentgroup' does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "user already in group",
			action: &AddUserToGroupAction{
				UserName:  "testuser",
				GroupName: "wheel",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "addgroup testuser wheel", errors.New("addgroup: user 'testuser' is already a member of 'wheel'"))
			},
			expectError: true,
			errorMsg:    "already a member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}

func TestRemoveUserFromGroupAction_Apply_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		action      *RemoveUserFromGroupAction
		setupFunc   func(*test.MockCommandRunner)
		expectError bool
		errorMsg    string
	}{
		{
			name: "empty username",
			action: &RemoveUserFromGroupAction{
				UserName:  "",
				GroupName: "wheel",
			},
			expectError: true,
		},
		{
			name: "empty group name",
			action: &RemoveUserFromGroupAction{
				UserName:  "testuser",
				GroupName: "",
			},
			expectError: true,
		},
		{
			name: "user not in group",
			action: &RemoveUserFromGroupAction{
				UserName:  "testuser",
				GroupName: "wheel",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "delgroup testuser wheel", errors.New("delgroup: user 'testuser' is not a member of 'wheel'"))
			},
			expectError: true,
			errorMsg:    "not a member",
		},
		{
			name: "user does not exist",
			action: &RemoveUserFromGroupAction{
				UserName:  "nonexistentuser",
				GroupName: "wheel",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "delgroup nonexistentuser wheel", errors.New("delgroup: user 'nonexistentuser' does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "group does not exist",
			action: &RemoveUserFromGroupAction{
				UserName:  "testuser",
				GroupName: "nonexistentgroup",
			},
			setupFunc: func(runner *test.MockCommandRunner) {
				runner.SetError("", "delgroup testuser nonexistentgroup", errors.New("delgroup: group 'nonexistentgroup' does not exist"))
			},
			expectError: true,
			errorMsg:    "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			runner := test.NewMockCommandRunner()
			if tt.setupFunc != nil {
				tt.setupFunc(runner)
			}

			logger := test.NewMockLogger(0)

			// Execute
			err := tt.action.Apply(runner, logger)

			// Assert
			if tt.expectError {
				require.Error(t, err, "Expected error for case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Expected no error for case: %s", tt.name)
			}
		})
	}
}
