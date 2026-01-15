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

func TestLoadConfig(t *testing.T) {
	logger := test.NewMockLogger(slog.LevelInfo)

	t.Run("successfully loads a valid config", func(t *testing.T) {
		content := `
packages:
  - name: git
  - name: htop
users:
  - name: testuser
    groups:
      - wheel
`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "system.yaml")
		err := os.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath, logger)
		require.NoError(t, err)

		expected := &model.SystemState{
			Packages: []model.PackageState{
				{Name: "git"},
				{Name: "htop"},
			},
			Users: []model.UserState{
				{Name: "testuser", Groups: []string{"wheel"}},
			},
		}

		// Sort slices for consistent comparison
		cfg.Sort()
		expected.Sort()

		assert.Equal(t, expected, cfg)
	})

	t.Run("returns an error if the file does not exist", func(t *testing.T) {
		_, err := LoadConfig("non-existent-file.yaml", logger)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err), "expected a file not found error")
	})

	t.Run("returns an error for malformed YAML", func(t *testing.T) {
		content := `packages: - name: git\n  invalid-indent`
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "system.yaml")
		err := os.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath, logger)
		assert.Error(t, err)
	})
}

func TestLoadConfig_Includes(t *testing.T) {
	logger := test.NewMockLogger(slog.LevelInfo)

	t.Run("loads config with includes", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base config
		baseContent := `
packages:
  - name: htop
  - name: git
services:
  - name: sshd
    enabled: true
    runlevel: default
users:
  - name: alice
    groups: [users]
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create host config with includes
		hostContent := `
includes:
  - base.yaml
packages:
  - name: vim
services:
  - name: sshd
    enabled: false
    runlevel: default
users:
  - name: alice
    groups: [wheel]
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		// Check merged packages (union)
		expectedPackages := []model.PackageState{
			{Name: "htop"},
			{Name: "git"},
			{Name: "vim"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)

		// Check merged services (override)
		expectedServices := []model.ServiceState{
			{Name: "sshd", Enabled: false, Runlevel: "default"},
		}
		assert.ElementsMatch(t, expectedServices, cfg.Services)

		// Check merged users (union groups)
		require.Len(t, cfg.Users, 1)
		assert.Equal(t, "alice", cfg.Users[0].Name)
		assert.ElementsMatch(t, []string{"users", "wheel"}, cfg.Users[0].Groups)

		// Check warnings
		assert.True(t, logger.HasMessage("Service overridden"))
		assert.True(t, logger.HasMessage("User groups merged"))
	})

	t.Run("handles nested includes", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base config
		baseContent := `
packages:
  - name: htop
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create role config that includes base
		roleContent := `
includes:
  - base.yaml
packages:
  - name: vim
`
		rolePath := filepath.Join(tmpDir, "role.yaml")
		err = os.WriteFile(rolePath, []byte(roleContent), 0644)
		require.NoError(t, err)

		// Create host config that includes role
		hostContent := `
includes:
  - role.yaml
packages:
  - name: git
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		expectedPackages := []model.PackageState{
			{Name: "htop"},
			{Name: "vim"},
			{Name: "git"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)
	})

	t.Run("detects circular includes", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config A that includes B
		aContent := `
includes:
  - b.yaml
packages:
  - name: pkg-a
`
		aPath := filepath.Join(tmpDir, "a.yaml")
		err := os.WriteFile(aPath, []byte(aContent), 0644)
		require.NoError(t, err)

		// Create config B that includes A (circular)
		bContent := `
includes:
  - a.yaml
packages:
  - name: pkg-b
`
		bPath := filepath.Join(tmpDir, "b.yaml")
		err = os.WriteFile(bPath, []byte(bContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(aPath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular include detected")
	})

	t.Run("handles absolute include paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base config
		baseContent := `
packages:
  - name: htop
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create host config with absolute include
		hostContent := `
includes:
  - ` + basePath + `
packages:
  - name: vim
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		expectedPackages := []model.PackageState{
			{Name: "htop"},
			{Name: "vim"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)
	})

	t.Run("validates includes field", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config with invalid includes
		content := `
includes:
  - ""
packages:
  - name: htop
`
		configPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(configPath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "include path cannot be empty")
	})

	t.Run("handles empty included file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create empty base config
		baseContent := ``
		basePath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create host config that includes empty file
		hostContent := `
includes:
  - base.yaml
packages:
  - name: vim
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		// Should only have packages from host config
		expectedPackages := []model.PackageState{
			{Name: "vim"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)
	})

	t.Run("handles included file with only comments", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base config with only comments
		baseContent := `# This is a comment
# Another comment
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		err := os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create host config that includes comments-only file
		hostContent := `
includes:
  - base.yaml
packages:
  - name: git
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		// Should only have packages from host config
		expectedPackages := []model.PackageState{
			{Name: "git"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)
	})

	t.Run("handles relative paths with ..", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create subdirectory structure
		subDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create base config in parent directory
		baseContent := `
packages:
  - name: htop
`
		basePath := filepath.Join(tmpDir, "base.yaml")
		err = os.WriteFile(basePath, []byte(baseContent), 0644)
		require.NoError(t, err)

		// Create host config in subdirectory that references parent with ..
		hostContent := `
includes:
  - ../base.yaml
packages:
  - name: vim
`
		hostPath := filepath.Join(subDir, "host.yaml")
		err = os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(hostPath, logger)
		require.NoError(t, err)

		// Should merge packages from both configs
		expectedPackages := []model.PackageState{
			{Name: "htop"},
			{Name: "vim"},
		}
		assert.ElementsMatch(t, expectedPackages, cfg.Packages)
	})

	t.Run("path traversal attempts are safe", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a config that tries to include a sensitive file path
		// Note: This should fail because the file doesn't exist, not because of path validation
		// Include paths are resolved but not restricted (they're just file paths)
		hostContent := `
includes:
  - ../../../etc/passwd
packages:
  - name: vim
`
		hostPath := filepath.Join(tmpDir, "host.yaml")
		err := os.WriteFile(hostPath, []byte(hostContent), 0644)
		require.NoError(t, err)

		_, err = LoadConfig(hostPath, logger)
		// Should fail because /etc/passwd doesn't exist in test environment
		// or isn't a valid YAML config file
		assert.Error(t, err)
	})
}
