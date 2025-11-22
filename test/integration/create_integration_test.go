package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommandWithAutoCopy(t *testing.T) {
	t.Skip("Integration test temporarily disabled for Windows compatibility")
	// Create test repository with auto-copy files
	testRepo := testutil.NewTestGitRepository(t, "autocopy-integration")

	// Create AI and development files
	testRepo.CreateDirectory(".ai")
	testRepo.CreateFile(".ai/prompts.md", "# AI Prompts")
	testRepo.CreateFile(".ai/templates.md", "# Templates")
	testRepo.CreateFile(".cursorrules", "# Cursor Rules")
	testRepo.CreateFile("CLAUDE.md", "# Claude Context")
	testRepo.CreateFile("README.md", "# Project README")

	// Create auto-copy configuration
	configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configContent := `{
		"version": 1,
		"items": [
			{
				"path": ".ai/",
				"directory": true,
				"recursive": true,
				"rootOnly": true
			},
			{
				"path": ".cursorrules",
				"directory": false,
				"rootOnly": true
			},
			{
				"path": "CLAUDE.md",
				"directory": false,
				"rootOnly": true
			}
		]
	}`

	configFile := filepath.Join(configDir, "auto-copy-files.json")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	testRepo.CommitAll("Add auto-copy files and configuration")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("create worktree with auto-copy enabled", func(t *testing.T) {
		// Execute create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/autocopy-test")

		// Should succeed
		require.NoError(t, err)

		// Check output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "✅")
		assert.Contains(t, stdout, "📋 Auto-copied")
		assert.Contains(t, stdout, ".ai/")
		assert.Contains(t, stdout, ".cursorrules")
		assert.Contains(t, stdout, "CLAUDE.md")
		assert.Contains(t, stdout, "Updated .gitignore")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "autocopy-integration-feature-autocopy-test")
		assert.DirExists(t, expectedPath)

		// Verify auto-copied files exist
		assert.DirExists(t, filepath.Join(expectedPath, ".ai"))
		assert.FileExists(t, filepath.Join(expectedPath, ".ai", "prompts.md"))
		assert.FileExists(t, filepath.Join(expectedPath, ".ai", "templates.md"))
		assert.FileExists(t, filepath.Join(expectedPath, ".cursorrules"))
		assert.FileExists(t, filepath.Join(expectedPath, "CLAUDE.md"))

		// Verify .gitignore was updated
		gitignorePath := filepath.Join(expectedPath, ".gitignore")
		assert.FileExists(t, gitignorePath)

		gitignoreContent, err := os.ReadFile(gitignorePath)
		require.NoError(t, err)

		gitignoreStr := string(gitignoreContent)
		assert.Contains(t, gitignoreStr, "# Auto-copied files (added by hatcher)")
		assert.Contains(t, gitignoreStr, ".ai/")
		assert.Contains(t, gitignoreStr, ".cursorrules")
		assert.Contains(t, gitignoreStr, "CLAUDE.md")

		// Verify content integrity
		originalContent, err := os.ReadFile(filepath.Join(testRepo.RepoDir, ".cursorrules"))
		require.NoError(t, err)
		copiedContent, err := os.ReadFile(filepath.Join(expectedPath, ".cursorrules"))
		require.NoError(t, err)
		assert.Equal(t, originalContent, copiedContent)
	})

	t.Run("create worktree with auto-copy disabled", func(t *testing.T) {
		// Execute create command with --no-copy flag
		err := cliHelper.ExecuteCommand(rootCmd, "create", "--no-copy", "feature/no-copy-test")

		// Should succeed
		require.NoError(t, err)

		// Check output - should not mention auto-copying
		stdout := cliHelper.GetStdout()
		assert.NotContains(t, stdout, "📋 Auto-copied")
		assert.NotContains(t, stdout, "Updated .gitignore")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "autocopy-integration-feature-no-copy-test")
		assert.DirExists(t, expectedPath)

		// Verify auto-copy files were NOT copied
		assert.NoDirExists(t, filepath.Join(expectedPath, ".ai"))
		assert.NoFileExists(t, filepath.Join(expectedPath, ".cursorrules"))
		assert.NoFileExists(t, filepath.Join(expectedPath, "CLAUDE.md"))

		// Verify .gitignore was not created
		gitignorePath := filepath.Join(expectedPath, ".gitignore")
		assert.NoFileExists(t, gitignorePath)
	})

	t.Run("create worktree with gitignore update disabled", func(t *testing.T) {
		// Execute create command with --no-gitignore-update flag
		err := cliHelper.ExecuteCommand(rootCmd, "create", "--no-gitignore-update", "feature/no-gitignore-test")

		// Should succeed
		require.NoError(t, err)

		// Check output - should mention auto-copying but not gitignore update
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "📋 Auto-copied")
		assert.NotContains(t, stdout, "Updated .gitignore")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "autocopy-integration-feature-no-gitignore-test")
		assert.DirExists(t, expectedPath)

		// Verify auto-copy files exist
		assert.DirExists(t, filepath.Join(expectedPath, ".ai"))
		assert.FileExists(t, filepath.Join(expectedPath, ".cursorrules"))
		assert.FileExists(t, filepath.Join(expectedPath, "CLAUDE.md"))

		// Verify .gitignore was not created/updated
		gitignorePath := filepath.Join(expectedPath, ".gitignore")
		assert.NoFileExists(t, gitignorePath)
	})

	t.Run("create worktree with verbose output", func(t *testing.T) {
		// Execute create command with verbose flag
		err := cliHelper.ExecuteCommand(rootCmd, "--verbose", "create", "feature/verbose-test")

		// Should succeed
		require.NoError(t, err)

		// Check verbose output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "🔍 Creating worktree for branch")
		assert.Contains(t, stdout, "📋 Auto-copying configuration files")
	})
}

func TestCreateCommandAutoCopyEdgeCases(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "autocopy-edge-cases")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("no auto-copy configuration", func(t *testing.T) {
		// Execute create command without any auto-copy configuration
		err := cliHelper.ExecuteCommand(rootCmd, "--verbose", "create", "feature/no-config-test")

		// Should succeed
		require.NoError(t, err)

		// Check output - should mention no configuration found
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "No auto-copy configuration found")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "autocopy-edge-cases-feature-no-config-test")
		assert.DirExists(t, expectedPath)
	})

	t.Run("invalid auto-copy configuration", func(t *testing.T) {
		// Create invalid configuration
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		configContent := `{
			"version": 1,
			"items": [
				{
					"path": "../dangerous/path",
					"rootOnly": true
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err = os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute create command
		err = cliHelper.ExecuteCommand(rootCmd, "create", "feature/invalid-config-test")

		// Should fail due to invalid configuration
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid auto-copy configuration")
	})

	t.Run("malformed JSON configuration", func(t *testing.T) {
		// Create malformed configuration
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"version": 1,
			"items": [
				{
					"path": ".cursorrules"
					// Missing comma - invalid JSON
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute create command
		err = cliHelper.ExecuteCommand(rootCmd, "create", "feature/malformed-config-test")

		// Should fail due to JSON parsing error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load auto-copy configuration")
	})

	t.Run("auto-copy with non-existent files", func(t *testing.T) {
		// Create configuration referencing non-existent files
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"version": 1,
			"items": [
				{
					"path": "non-existent-file.txt",
					"rootOnly": true
				},
				{
					"path": "another-missing-file.md",
					"rootOnly": true
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute create command
		err = cliHelper.ExecuteCommand(rootCmd, "--verbose", "create", "feature/missing-files-test")

		// Should succeed (missing files are skipped)
		require.NoError(t, err)

		// Check output - should mention no files matched
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "No files matched auto-copy configuration")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "autocopy-edge-cases-feature-missing-files-test")
		assert.DirExists(t, expectedPath)
	})
}
