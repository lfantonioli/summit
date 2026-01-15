package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"summit/pkg/test"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ErrorCases(t *testing.T) {
	logger := test.NewMockLogger(slog.LevelInfo)

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "file does not exist",
			configYAML:  "",
			expectError: true,
			errorMsg:    "no such file",
		},
		{
			name: "malformed YAML - unclosed quote",
			configYAML: `packages:
  - name: "htop
services:
  - name: nginx
`,
			expectError: true,
		},
		{
			name: "invalid YAML structure - array instead of object",
			configYAML: `- packages:
  - name: htop
- services:
  - name: nginx
`,
			expectError: true,
		},
		{
			name: "invalid YAML - duplicate keys",
			configYAML: `packages:
  - name: htop
packages:
  - name: vim
`,
			expectError: true,
		},
		{
			name: "invalid YAML - bad anchor",
			configYAML: `packages: &anchor
  - name: htop
services: *badanchor
`,
			expectError: true,
		},
		{
			name: "malformed YAML - unclosed quote",
			configYAML: `packages:
  - name: "htop
services:
  - name: nginx
`,
			expectError: true,
		},

		{
			name: "invalid YAML structure - array instead of object",
			configYAML: `- packages:
  - name: htop
- services:
  - name: nginx
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")

			if tt.configYAML != "" {
				err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
				require.NoError(t, err)
			} else if tt.name == "file does not exist" {
				configPath = "/nonexistent/file.yaml"
			} else {
				// Empty file
				err := os.WriteFile(configPath, []byte(""), 0644)
				require.NoError(t, err)
			}

			// Execute
			_, err := LoadConfig(configPath, logger)

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
