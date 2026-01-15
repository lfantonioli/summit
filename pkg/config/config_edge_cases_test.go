package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"summit/pkg/model"
	"summit/pkg/test"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_TableDrivenEdgeCases(t *testing.T) {
	logger := test.NewMockLogger(slog.LevelInfo)

	tests := []struct {
		name       string
		configYAML string
		validate   func(t *testing.T, cfg *model.SystemState)
	}{
		{
			name: "empty sections",
			configYAML: `packages: []
services: []
users: []
configs: []
user-packages: []
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				assert.Empty(t, cfg.Packages)
				assert.Empty(t, cfg.Services)
				assert.Empty(t, cfg.Users)
				assert.Empty(t, cfg.Configs)
				assert.Empty(t, cfg.UserPackages)
			},
		},
		{
			name: "minimal valid config",
			configYAML: `packages:
  - name: htop
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Packages, 1)
				assert.Equal(t, "htop", cfg.Packages[0].Name)
			},
		},
		{
			name: "special characters in content",
			configYAML: `configs:
  - path: /etc/motd
    content: |
      Welcome!
      Special chars: àáâãäå
      Quotes: "single' and "double"
      Newlines and tabs:
      	indented
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Configs, 1)
				assert.Contains(t, cfg.Configs[0].Content, "Special chars: àáâãäå")
				assert.Contains(t, cfg.Configs[0].Content, "Quotes: \"single' and \"double\"")
			},
		},
		{
			name: "nested structures",
			configYAML: `users:
  - name: testuser
    groups:
      - wheel
      - docker
user-packages:
  - user: testuser
    pipx:
      - black
      - ruff
    npm:
      - typescript
      - eslint
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Users, 1)
				assert.Equal(t, "testuser", cfg.Users[0].Name)
				assert.Contains(t, cfg.Users[0].Groups, "wheel")
				assert.Contains(t, cfg.Users[0].Groups, "docker")

				require.Len(t, cfg.UserPackages, 1)
				assert.Equal(t, "testuser", cfg.UserPackages[0].User)
				assert.Contains(t, cfg.UserPackages[0].Pipx, "black")
				assert.Contains(t, cfg.UserPackages[0].Npm, "typescript")
			},
		},
		{
			name: "complex service configuration",
			configYAML: `services:
  - name: nginx
    enabled: true
    runlevel: default
  - name: sshd
    enabled: false
    runlevel: boot
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Services, 2)
				assert.Equal(t, "nginx", cfg.Services[0].Name)
				assert.True(t, cfg.Services[0].Enabled)
				assert.Equal(t, "default", cfg.Services[0].Runlevel)
				assert.Equal(t, "sshd", cfg.Services[1].Name)
				assert.False(t, cfg.Services[1].Enabled)
				assert.Equal(t, "boot", cfg.Services[1].Runlevel)
			},
		},
		{
			name: "configs with all fields",
			configYAML: `configs:
  - path: /etc/nginx/nginx.conf
    content: |
      user nginx;
      worker_processes 1;
    mode: "0644"
    owner: root
    group: root
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Configs, 1)
				config := cfg.Configs[0]
				assert.Equal(t, "/etc/nginx/nginx.conf", config.Path)
				assert.Contains(t, config.Content, "user nginx;")
				assert.Equal(t, "0644", config.Mode)
				assert.Equal(t, "root", config.Owner)
				assert.Equal(t, "root", config.Group)
			},
		},
		{
			name: "ignored configs",
			configYAML: `ignored-configs:
  - "*.bak"
  - "/etc/hostname"
configs:
  - path: /etc/test.conf
    content: "test"
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				assert.Contains(t, cfg.IgnoredConfigs, "*.bak")
				assert.Contains(t, cfg.IgnoredConfigs, "/etc/hostname")
				require.Len(t, cfg.Configs, 1)
			},
		},
		{
			name: "very long package names",
			configYAML: `packages:
  - name: very-long-package-name-that-might-cause-issues-with-path-lengths-or-other-limits-in-the-system
  - name: another-extremely-long-package-name-that-tests-boundaries-and-edge-cases-in-the-configuration-parsing
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Packages, 2)
				// Packages are sorted alphabetically
				assert.Equal(t, "another-extremely-long-package-name-that-tests-boundaries-and-edge-cases-in-the-configuration-parsing", cfg.Packages[0].Name)
				assert.Equal(t, "very-long-package-name-that-might-cause-issues-with-path-lengths-or-other-limits-in-the-system", cfg.Packages[1].Name)
			},
		},
		{
			name: "unicode in package names",
			configYAML: `packages:
  - name: package_with_üñíçødé
  - name: 包名
`,
			validate: func(t *testing.T, cfg *model.SystemState) {
				require.Len(t, cfg.Packages, 2)
				assert.Equal(t, "package_with_üñíçødé", cfg.Packages[0].Name)
				assert.Equal(t, "包名", cfg.Packages[1].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			require.NoError(t, err)

			// Execute
			cfg, err := LoadConfig(configPath, logger)
			require.NoError(t, err)

			// Validate
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}
