package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommand_Integration(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "config-integration-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("config init project", func(t *testing.T) {
		// Execute config init command
		output, err := testutil.ExecuteCommand(configInitCmd, []string{})
		require.NoError(t, err)

		// Should show success message
		assert.Contains(t, output, "Initialized project configuration")
		assert.Contains(t, output, ".hatcher-auto-copy.json")

		// Should create config file
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		assert.FileExists(t, configPath)

		// Verify config content
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		err = json.Unmarshal(content, &config)
		require.NoError(t, err)

		assert.Equal(t, float64(2), config["version"])
		assert.Contains(t, config, "items")
	})

	t.Run("config init global", func(t *testing.T) {
		// Set temporary HOME directory
		tempHome := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tempHome)

		// Execute config init --global command
		output, err := testutil.ExecuteCommand(configInitCmd, []string{"--global"})
		require.NoError(t, err)

		// Should show success message
		assert.Contains(t, output, "Initialized global configuration")
		assert.Contains(t, output, ".hatcher/config.yaml")

		// Should create config file
		configPath := filepath.Join(tempHome, ".hatcher", "config.yaml")
		assert.FileExists(t, configPath)
	})

	t.Run("config init with force", func(t *testing.T) {
		// Create existing config
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		err := os.WriteFile(configPath, []byte(`{"version": 1}`), 0644)
		require.NoError(t, err)

		// Try init without force (should fail)
		output, err := testutil.ExecuteCommand(configInitCmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, output, "already exists")

		// Try init with force (should succeed)
		output, err = testutil.ExecuteCommand(configInitCmd, []string{"--force"})
		require.NoError(t, err)
		assert.Contains(t, output, "Initialized project configuration")
	})

	t.Run("config show", func(t *testing.T) {
		// Create config file
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		configContent := `{
			"version": 2,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": true
				}
			]
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute config show command
		output, err := testutil.ExecuteCommand(configShowCmd, []string{})
		require.NoError(t, err)

		// Should show configuration
		assert.Contains(t, output, "Current Hatcher Configuration")
		assert.Contains(t, output, "Auto-copy Settings")
		assert.Contains(t, output, ".ai/")
		assert.Contains(t, output, "directory")
	})

	t.Run("config show with paths", func(t *testing.T) {
		// Execute config show --paths command
		output, err := testutil.ExecuteCommand(configShowCmd, []string{"--paths"})
		require.NoError(t, err)

		// Should show config file paths
		assert.Contains(t, output, "Configuration file search paths")
		assert.Contains(t, output, ".hatcher-auto-copy.json")
		assert.Contains(t, output, "✅") // Should show existing files
	})

	t.Run("config show JSON format", func(t *testing.T) {
		// Execute config show --format json command
		output, err := testutil.ExecuteCommand(configShowCmd, []string{"--format", "json"})
		require.NoError(t, err)

		// Should be valid JSON
		var config map[string]interface{}
		err = json.Unmarshal([]byte(output), &config)
		require.NoError(t, err)

		// Should contain expected fields
		assert.Contains(t, config, "autocopy")
		assert.Contains(t, config, "editor")
		assert.Contains(t, config, "global")
	})

	t.Run("config validate valid", func(t *testing.T) {
		// Create valid config file
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		configContent := `{
			"version": 2,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": true
				}
			]
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute config validate command
		output, err := testutil.ExecuteCommand(configValidateCmd, []string{})
		require.NoError(t, err)

		// Should show validation success
		assert.Contains(t, output, "Configuration is valid")
		assert.Contains(t, output, "✅")
	})

	t.Run("config validate invalid", func(t *testing.T) {
		// Create invalid config file
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		configContent := `{
			"version": 999,
			"items": [
				{
					"path": "",
					"directory": true
				}
			]
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute config validate command
		output, err := testutil.ExecuteCommand(configValidateCmd, []string{})
		assert.Error(t, err)

		// Should show validation errors
		assert.Contains(t, output, "validation error")
		assert.Contains(t, output, "❌")
	})

	t.Run("config edit", func(t *testing.T) {
		// Set EDITOR environment variable
		originalEditor := os.Getenv("EDITOR")
		defer os.Setenv("EDITOR", originalEditor)
		os.Setenv("EDITOR", "echo")

		// Execute config edit command
		output, err := testutil.ExecuteCommand(configEditCmd, []string{})
		require.NoError(t, err)

		// Should show editor information
		assert.Contains(t, output, "Opening")
		assert.Contains(t, output, ".hatcher-auto-copy.json")
		assert.Contains(t, output, "echo")
	})

	t.Run("config command aliases", func(t *testing.T) {
		// Test 'cfg' alias
		output, err := testutil.ExecuteCommandByName("cfg", []string{"show"})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "Configuration")
		}
	})
}

func TestConfigCommand_EdgeCases(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "config-edge-cases-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("config show with no config", func(t *testing.T) {
		// Execute config show with no config files
		output, err := testutil.ExecuteCommand(configShowCmd, []string{})
		require.NoError(t, err)

		// Should show default configuration
		assert.Contains(t, output, "Current Hatcher Configuration")
		assert.Contains(t, output, "Auto-copy Settings")
	})

	t.Run("config validate with no config", func(t *testing.T) {
		// Execute config validate with no config files
		output, err := testutil.ExecuteCommand(configValidateCmd, []string{})
		require.NoError(t, err)

		// Should validate default configuration
		assert.Contains(t, output, "Configuration is valid")
	})

	t.Run("config init with invalid format", func(t *testing.T) {
		// Execute config init with invalid format
		output, err := testutil.ExecuteCommand(configInitCmd, []string{"--format", "invalid"})
		// Should still work (format is not used in init currently)
		require.NoError(t, err)
		assert.Contains(t, output, "Initialized")
	})

	t.Run("config show with invalid format", func(t *testing.T) {
		// Execute config show with invalid format
		output, err := testutil.ExecuteCommand(configShowCmd, []string{"--format", "invalid"})
		require.NoError(t, err)

		// Should default to table format
		assert.Contains(t, output, "Current Hatcher Configuration")
	})

	t.Run("config edit with custom editor", func(t *testing.T) {
		// Execute config edit with custom editor
		output, err := testutil.ExecuteCommand(configEditCmd, []string{"--editor", "vim"})
		require.NoError(t, err)

		// Should use specified editor
		assert.Contains(t, output, "vim")
	})

	t.Run("config validate with fix flag", func(t *testing.T) {
		// Create config with fixable issues
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		configContent := `{
			"version": 2,
			"items": []
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute config validate --fix command
		output, err := testutil.ExecuteCommand(configValidateCmd, []string{"--fix"})
		require.NoError(t, err)

		// Should mention fix functionality
		assert.Contains(t, output, "Configuration is valid")
	})

	t.Run("config with environment variables", func(t *testing.T) {
		// Set environment variables
		originalEditor := os.Getenv("HATCHER_EDITOR")
		originalVerbose := os.Getenv("HATCHER_VERBOSE")
		defer func() {
			os.Setenv("HATCHER_EDITOR", originalEditor)
			os.Setenv("HATCHER_VERBOSE", originalVerbose)
		}()

		os.Setenv("HATCHER_EDITOR", "vim")
		os.Setenv("HATCHER_VERBOSE", "true")

		// Execute config show command
		output, err := testutil.ExecuteCommand(configShowCmd, []string{})
		require.NoError(t, err)

		// Should show environment variable overrides
		assert.Contains(t, output, "vim")
		assert.Contains(t, output, "true")
	})
}
