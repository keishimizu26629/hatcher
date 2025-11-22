package autocopy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoCopyConfig_LoadFromFile(t *testing.T) {
	t.Run("load new format config", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "auto-copy-files.json")

		configContent := `{
			"version": 1,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": false,
					"rootOnly": true
				},
				{
					"path": ".cursorrules",
					"autoDetect": true,
					"recursive": false,
					"rootOnly": true
				},
				{
					"path": "CLAUDE.md",
					"directory": false,
					"rootOnly": true
				}
			]
		}`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test loading
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Verify config content
		assert.Equal(t, 1, config.Version)
		assert.Len(t, config.Items, 3)

		// Check first item (.ai/)
		item1 := config.Items[0]
		assert.Equal(t, ".ai/", item1.Path)
		assert.True(t, *item1.Directory)
		assert.False(t, item1.Recursive)
		assert.True(t, item1.RootOnly)
		assert.False(t, item1.AutoDetect)

		// Check second item (.cursorrules with autoDetect)
		item2 := config.Items[1]
		assert.Equal(t, ".cursorrules", item2.Path)
		assert.Nil(t, item2.Directory) // Should be nil when autoDetect is true
		assert.False(t, item2.Recursive)
		assert.True(t, item2.RootOnly)
		assert.True(t, item2.AutoDetect)

		// Check third item (CLAUDE.md)
		item3 := config.Items[2]
		assert.Equal(t, "CLAUDE.md", item3.Path)
		assert.False(t, *item3.Directory)
		assert.False(t, item3.Recursive)
		assert.True(t, item3.RootOnly)
		assert.False(t, item3.AutoDetect)
	})

	t.Run("load legacy format config", func(t *testing.T) {
		// Create temporary config file with legacy format
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "auto-copy-files.json")

		configContent := `{
			"files": [
				".ai/",
				".cursorrules",
				"CLAUDE.md"
			]
		}`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test loading
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Verify legacy format is converted to new format
		assert.Equal(t, 0, config.Version) // Legacy format has version 0
		assert.Len(t, config.Files, 3)
		assert.Contains(t, config.Files, ".ai/")
		assert.Contains(t, config.Files, ".cursorrules")
		assert.Contains(t, config.Files, "CLAUDE.md")
	})

	t.Run("config file priority", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create multiple config files
		highPriorityConfig := filepath.Join(tempDir, "high-priority.json")
		lowPriorityConfig := filepath.Join(tempDir, "low-priority.json")

		highPriorityContent := `{
			"version": 1,
			"items": [
				{
					"path": "high-priority-file.txt",
					"directory": false,
					"rootOnly": true
				}
			]
		}`

		lowPriorityContent := `{
			"version": 1,
			"items": [
				{
					"path": "low-priority-file.txt",
					"directory": false,
					"rootOnly": true
				}
			]
		}`

		err := os.WriteFile(highPriorityConfig, []byte(highPriorityContent), 0644)
		require.NoError(t, err)
		err = os.WriteFile(lowPriorityConfig, []byte(lowPriorityContent), 0644)
		require.NoError(t, err)

		// Test priority order (first file should be used)
		config, err := LoadAutoCopyConfig([]string{highPriorityConfig, lowPriorityConfig})
		require.NoError(t, err)

		assert.Len(t, config.Items, 1)
		assert.Equal(t, "high-priority-file.txt", config.Items[0].Path)
	})

	t.Run("no config file found", func(t *testing.T) {
		// Test with non-existent files
		config, err := LoadAutoCopyConfig([]string{"/non/existent/file.json"})
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Should return empty config
		assert.Equal(t, 0, config.Version)
		assert.Empty(t, config.Items)
		assert.Empty(t, config.Files)
	})

	t.Run("invalid JSON format", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid.json")

		// Write invalid JSON
		err := os.WriteFile(configFile, []byte("invalid json content"), 0644)
		require.NoError(t, err)

		// Should return error
		config, err := LoadAutoCopyConfig([]string{configFile})
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

func TestAutoCopyConfig_Validate(t *testing.T) {
	t.Run("valid new format config", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:      ".ai/",
					Directory: testutil.BoolPtr(true),
					RootOnly:  true,
				},
				{
					Path:       ".cursorrules",
					AutoDetect: true,
					RootOnly:   true,
				},
			},
		}

		err := ValidateAutoCopyConfig(config)
		assert.NoError(t, err)
	})

	t.Run("valid legacy format config", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 0,
			Files:   []string{".ai/", ".cursorrules", "CLAUDE.md"},
		}

		err := ValidateAutoCopyConfig(config)
		assert.NoError(t, err)
	})

	t.Run("invalid config - empty path", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:     "", // Empty path
					RootOnly: true,
				},
			},
		}

		err := ValidateAutoCopyConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path cannot be empty")
	})

	t.Run("invalid config - dangerous path", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:     "../dangerous/path",
					RootOnly: true,
				},
			},
		}

		err := ValidateAutoCopyConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous path")
	})

	t.Run("invalid config - conflicting options", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:       ".cursorrules",
					Directory:  testutil.BoolPtr(true),
					AutoDetect: true, // Conflicting with explicit directory setting
					RootOnly:   true,
				},
			},
		}

		err := ValidateAutoCopyConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use both directory and autoDetect")
	})
}

func TestAutoCopyItem_IsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		item     AutoCopyItem
		expected bool
	}{
		{
			name: "explicit directory true",
			item: AutoCopyItem{
				Path:      ".ai/",
				Directory: testutil.BoolPtr(true),
			},
			expected: true,
		},
		{
			name: "explicit directory false",
			item: AutoCopyItem{
				Path:      "file.txt",
				Directory: testutil.BoolPtr(false),
			},
			expected: false,
		},
		{
			name: "auto-detect from path ending with slash",
			item: AutoCopyItem{
				Path: ".ai/",
			},
			expected: true,
		},
		{
			name: "auto-detect from path not ending with slash",
			item: AutoCopyItem{
				Path: "file.txt",
			},
			expected: false,
		},
		{
			name: "autoDetect enabled - should return false as default",
			item: AutoCopyItem{
				Path:       ".cursorrules",
				AutoDetect: true,
			},
			expected: false, // Default when autoDetect is enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsDirectory()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoCopyItem_IsGlobPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "simple path",
			path:     ".cursorrules",
			expected: false,
		},
		{
			name:     "path with asterisk",
			path:     "*.json",
			expected: true,
		},
		{
			name:     "path with question mark",
			path:     "file?.txt",
			expected: true,
		},
		{
			name:     "path with brackets",
			path:     "file[0-9].txt",
			expected: true,
		},
		{
			name:     "path with double asterisk",
			path:     "**/.cursorrules",
			expected: true,
		},
		{
			name:     "complex glob pattern",
			path:     "**/config/**/*.json",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := AutoCopyItem{Path: tt.path}
			result := item.IsGlobPattern()
			assert.Equal(t, tt.expected, result)
		})
	}
}
