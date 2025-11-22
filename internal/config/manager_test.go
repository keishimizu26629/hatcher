package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_LoadConfig(t *testing.T) {
	// Create temporary directory for test configs
	tempDir := t.TempDir()

	t.Run("load default config", func(t *testing.T) {
		manager := NewManager()

		config, err := manager.LoadConfig("")
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Should have default values
		assert.NotEmpty(t, config.AutoCopy.Items)
		assert.Equal(t, 2, config.AutoCopy.Version)
	})

	t.Run("load project config", func(t *testing.T) {
		// Create project config file
		projectConfigPath := filepath.Join(tempDir, ".hatcher-auto-copy.json")
		projectConfig := `{
			"version": 2,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": true
				},
				{
					"path": "custom-file.txt",
					"directory": false
				}
			]
		}`
		err := os.WriteFile(projectConfigPath, []byte(projectConfig), 0644)
		require.NoError(t, err)

		manager := NewManager()
		config, err := manager.LoadConfig(tempDir)
		require.NoError(t, err)

		// Should load project config
		assert.Len(t, config.AutoCopy.Items, 2)
		assert.Equal(t, ".ai/", config.AutoCopy.Items[0].Path)
		assert.Equal(t, "custom-file.txt", config.AutoCopy.Items[1].Path)
	})

	t.Run("load global config", func(t *testing.T) {
		// Create global config directory
		globalConfigDir := filepath.Join(tempDir, ".hatcher")
		err := os.MkdirAll(globalConfigDir, 0755)
		require.NoError(t, err)

		globalConfigPath := filepath.Join(globalConfigDir, "config.yaml")
		globalConfig := `
autocopy:
  version: 2
  items:
    - path: ".cursorrules"
      directory: false
    - path: "global-dir/"
      directory: true
      recursive: true
editor:
  preferred: "cursor"
  autoSwitch: true
`
		err = os.WriteFile(globalConfigPath, []byte(globalConfig), 0644)
		require.NoError(t, err)

		// Set HOME to temp directory
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tempDir)

		manager := NewManager()
		config, err := manager.LoadConfig("")
		require.NoError(t, err)

		// Should load global config
		assert.Equal(t, "cursor", config.Editor.Preferred)
		assert.True(t, config.Editor.AutoSwitch)
		assert.Len(t, config.AutoCopy.Items, 2)
	})

	t.Run("config priority order", func(t *testing.T) {
		// Create both global and project configs
		globalConfigDir := filepath.Join(tempDir, ".hatcher")
		err := os.MkdirAll(globalConfigDir, 0755)
		require.NoError(t, err)

		globalConfigPath := filepath.Join(globalConfigDir, "config.yaml")
		globalConfig := `
autocopy:
  version: 2
  items:
    - path: "global-file.txt"
      directory: false
editor:
  preferred: "code"
`
		err = os.WriteFile(globalConfigPath, []byte(globalConfig), 0644)
		require.NoError(t, err)

		projectConfigPath := filepath.Join(tempDir, ".hatcher-auto-copy.json")
		projectConfig := `{
			"version": 2,
			"items": [
				{
					"path": "project-file.txt",
					"directory": false
				}
			]
		}`
		err = os.WriteFile(projectConfigPath, []byte(projectConfig), 0644)
		require.NoError(t, err)

		// Set HOME to temp directory
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tempDir)

		manager := NewManager()
		config, err := manager.LoadConfig(tempDir)
		require.NoError(t, err)

		// Project config should override global for autocopy
		assert.Len(t, config.AutoCopy.Items, 1)
		assert.Equal(t, "project-file.txt", config.AutoCopy.Items[0].Path)

		// Global config should provide editor settings
		assert.Equal(t, "code", config.Editor.Preferred)
	})

	t.Run("environment variable override", func(t *testing.T) {
		// Set environment variables
		originalEditor := os.Getenv("HATCHER_EDITOR")
		originalVerbose := os.Getenv("HATCHER_VERBOSE")
		defer func() {
			os.Setenv("HATCHER_EDITOR", originalEditor)
			os.Setenv("HATCHER_VERBOSE", originalVerbose)
		}()

		os.Setenv("HATCHER_EDITOR", "vim")
		os.Setenv("HATCHER_VERBOSE", "true")

		manager := NewManager()
		config, err := manager.LoadConfig("")
		require.NoError(t, err)

		// Environment variables should override config
		assert.Equal(t, "vim", config.Editor.Preferred)
		assert.True(t, config.Global.Verbose)
	})
}

func TestManager_SaveConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("save project config", func(t *testing.T) {
		config := &Config{
			AutoCopy: AutoCopyConfig{
				Version: 2,
				Items: []AutoCopyItem{
					{
						Path:      ".ai/",
						Directory: testutil.BoolPtr(true),
						Recursive: true,
					},
				},
			},
		}

		manager := NewManager()
		err := manager.SaveConfig(config, tempDir, false)
		require.NoError(t, err)

		// Check if file was created
		configPath := filepath.Join(tempDir, ".hatcher-auto-copy.json")
		assert.FileExists(t, configPath)

		// Verify content
		savedConfig, err := manager.LoadConfig(tempDir)
		require.NoError(t, err)
		assert.Len(t, savedConfig.AutoCopy.Items, 1)
		assert.Equal(t, ".ai/", savedConfig.AutoCopy.Items[0].Path)
	})

	t.Run("save global config", func(t *testing.T) {
		config := &Config{
			Editor: EditorConfig{
				Preferred:  "cursor",
				AutoSwitch: true,
			},
			AutoCopy: AutoCopyConfig{
				Version: 2,
				Items: []AutoCopyItem{
					{
						Path:      ".cursorrules",
						Directory: testutil.BoolPtr(false),
					},
				},
			},
		}

		// Set HOME to temp directory
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tempDir)

		manager := NewManager()
		err := manager.SaveConfig(config, "", true)
		require.NoError(t, err)

		// Check if file was created
		configPath := filepath.Join(tempDir, ".hatcher", "config.yaml")
		assert.FileExists(t, configPath)

		// Verify content
		savedConfig, err := manager.LoadConfig("")
		require.NoError(t, err)
		assert.Equal(t, "cursor", savedConfig.Editor.Preferred)
		assert.True(t, savedConfig.Editor.AutoSwitch)
	})
}

func TestManager_ValidateConfig(t *testing.T) {
	manager := NewManager()

	t.Run("valid config", func(t *testing.T) {
		config := &Config{
			AutoCopy: AutoCopyConfig{
				Version: 2,
				Items: []AutoCopyItem{
					{
						Path:      ".ai/",
						Directory: testutil.BoolPtr(true),
						Recursive: true,
					},
				},
			},
		}

		errors := manager.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("invalid version", func(t *testing.T) {
		config := &Config{
			AutoCopy: AutoCopyConfig{
				Version: 999,
				Items: []AutoCopyItem{
					{
						Path: ".ai/",
					},
				},
			},
		}

		errors := manager.ValidateConfig(config)
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors[0], "unsupported version")
	})

	t.Run("empty path", func(t *testing.T) {
		config := &Config{
			AutoCopy: AutoCopyConfig{
				Version: 2,
				Items: []AutoCopyItem{
					{
						Path: "",
					},
				},
			},
		}

		errors := manager.ValidateConfig(config)
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors[0], "empty path")
	})

	t.Run("invalid editor", func(t *testing.T) {
		config := &Config{
			Editor: EditorConfig{
				Preferred: "invalid-editor",
			},
		}

		errors := manager.ValidateConfig(config)
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors[0], "unsupported editor")
	})
}

func TestManager_MigrateConfig(t *testing.T) {
	manager := NewManager()

	t.Run("migrate from v1 to v2", func(t *testing.T) {
		v1Config := map[string]interface{}{
			"version": 1,
			"files": []string{
				".ai/",
				".cursorrules",
				"CLAUDE.md",
			},
		}

		v2Config, err := manager.MigrateConfig(v1Config)
		require.NoError(t, err)
		assert.Equal(t, 2, v2Config.AutoCopy.Version)
		assert.Len(t, v2Config.AutoCopy.Items, 3)

		// Check migrated items
		assert.Equal(t, ".ai/", v2Config.AutoCopy.Items[0].Path)
		assert.True(t, *v2Config.AutoCopy.Items[0].Directory) // Should detect directory
		assert.True(t, v2Config.AutoCopy.Items[0].AutoDetect)

		assert.Equal(t, ".cursorrules", v2Config.AutoCopy.Items[1].Path)
		assert.False(t, *v2Config.AutoCopy.Items[1].Directory) // Should detect file
	})

	t.Run("no migration needed", func(t *testing.T) {
		v2Config := map[string]interface{}{
			"version": 2,
			"items": []map[string]interface{}{
				{
					"path":      ".ai/",
					"directory": true,
					"recursive": true,
				},
			},
		}

		config, err := manager.MigrateConfig(v2Config)
		require.NoError(t, err)
		assert.Equal(t, 2, config.AutoCopy.Version)
		assert.Len(t, config.AutoCopy.Items, 1)
	})
}

func TestManager_GetConfigPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	t.Run("get project config paths", func(t *testing.T) {
		paths := manager.GetConfigPaths(tempDir)

		expected := []string{
			filepath.Join(tempDir, ".hatcher-auto-copy.json"),
			filepath.Join(tempDir, ".hatcher-auto-copy.yaml"),
			filepath.Join(tempDir, ".hatcher", "config.json"),
			filepath.Join(tempDir, ".hatcher", "config.yaml"),
		}

		assert.Equal(t, expected, paths)
	})

	t.Run("get global config paths", func(t *testing.T) {
		paths := manager.GetConfigPaths("")

		expected := []string{
			filepath.Join(tempDir, ".hatcher", "config.json"),
			filepath.Join(tempDir, ".hatcher", "config.yaml"),
		}

		assert.Equal(t, expected, paths)
	})
}
