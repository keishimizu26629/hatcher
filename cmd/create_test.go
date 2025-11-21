package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommand(t *testing.T) {
	// Create a test Git repository
	testRepo := helpers.NewTestGitRepository(t, "test-project")

	// Create CLI test helper
	cliHelper := helpers.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := helpers.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("create worktree for new branch", func(t *testing.T) {
		// Execute create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/test-branch")

		// Should succeed
		require.NoError(t, err)

		// Check output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "🔍 Creating worktree for branch 'feature/test-branch'")
		assert.Contains(t, stdout, "✅")
		assert.Contains(t, stdout, "test-project-feature-test-branch")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "test-project-feature-test-branch")
		assert.DirExists(t, expectedPath)
	})

	t.Run("create worktree with dry-run", func(t *testing.T) {
		// Execute create command with dry-run
		err := cliHelper.ExecuteCommand(rootCmd, "create", "--dry-run", "feature/dry-run-test")

		// Should succeed
		require.NoError(t, err)

		// Check output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "🔍 Dry run mode")
		assert.Contains(t, stdout, "Would create worktree")

		// Verify worktree was NOT created
		expectedPath := filepath.Join(testRepo.TempDir, "test-project-feature-dry-run-test")
		assert.NoDirExists(t, expectedPath)
	})

	t.Run("create worktree with invalid branch name", func(t *testing.T) {
		// Execute create command with invalid branch name
		err := cliHelper.ExecuteCommand(rootCmd, "create", "invalid/../branch")

		// Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid branch name")
	})

	t.Run("create worktree with force flag", func(t *testing.T) {
		branchName := "feature/force-test"
		expectedPath := filepath.Join(testRepo.TempDir, "test-project-feature-force-test")

		// Create worktree first
		err := cliHelper.ExecuteCommand(rootCmd, "create", branchName)
		require.NoError(t, err)
		assert.DirExists(t, expectedPath)

		// Create a file in the worktree
		testFile := filepath.Join(expectedPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Try to create again with force
		err = cliHelper.ExecuteCommand(rootCmd, "create", "--force", branchName+"-2")
		require.NoError(t, err)

		// Should succeed and create new worktree
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "✅")
	})

	t.Run("create worktree with no-copy flag", func(t *testing.T) {
		// Execute create command with no-copy flag
		err := cliHelper.ExecuteCommand(rootCmd, "create", "--no-copy", "feature/no-copy-test")

		// Should succeed
		require.NoError(t, err)

		// Check output - should not mention auto-copying
		stdout := cliHelper.GetStdout()
		assert.NotContains(t, stdout, "📋 Auto-copying")
	})

	t.Run("create worktree outside git repository", func(t *testing.T) {
		// Change to a non-Git directory
		tempDir := t.TempDir()
		mockEnv.ChangeDir(tempDir)

		// Execute create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/test")

		// Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Not in a Git repository")

		// Change back to Git repository for cleanup
		mockEnv.ChangeDir(testRepo.RepoDir)
	})
}

func TestCreateCommandHelp(t *testing.T) {
	cliHelper := helpers.NewCLITestHelper(t)

	// Execute help command
	err := cliHelper.ExecuteCommand(rootCmd, "create", "--help")
	require.NoError(t, err)

	// Check help output
	stdout := cliHelper.GetStdout()
	assert.Contains(t, stdout, "Create a new worktree")
	assert.Contains(t, stdout, "--no-copy")
	assert.Contains(t, stdout, "--force")
	assert.Contains(t, stdout, "--dry-run")
}

func TestCreateCommandFlags(t *testing.T) {
	// Create a test Git repository
	testRepo := helpers.NewTestGitRepository(t, "test-project")

	// Create CLI test helper
	cliHelper := helpers.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := helpers.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("verbose flag", func(t *testing.T) {
		// Execute create command with verbose flag
		err := cliHelper.ExecuteCommand(rootCmd, "--verbose", "create", "feature/verbose-test")

		// Should succeed
		require.NoError(t, err)

		// Check verbose output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "🔍 Creating worktree for branch")
	})

	t.Run("combined flags", func(t *testing.T) {
		// Execute create command with multiple flags
		err := cliHelper.ExecuteCommand(rootCmd, "create", "--dry-run", "--no-copy", "--force", "feature/combined-test")

		// Should succeed
		require.NoError(t, err)

		// Check output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "🔍 Dry run mode")
		assert.NotContains(t, stdout, "📋 Auto-copying")
	})
}

func TestCreateCommandEdgeCase(t *testing.T) {
	// Create a test Git repository
	testRepo := helpers.NewTestGitRepository(t, "test-project")

	// Create CLI test helper
	cliHelper := helpers.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := helpers.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("branch name with special characters", func(t *testing.T) {
		// Execute create command with special characters in branch name
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/user@auth#2024")

		// Should succeed (special characters should be sanitized)
		require.NoError(t, err)

		// Check that worktree was created with sanitized name
		expectedPath := filepath.Join(testRepo.TempDir, "test-project-feature-user-auth-2024")
		assert.DirExists(t, expectedPath)
	})

	t.Run("very long branch name", func(t *testing.T) {
		// Create a very long branch name
		longBranchName := "feature/" + strings.Repeat("very-long-name", 10)

		// Execute create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", longBranchName)

		// Should fail due to validation
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("empty branch name", func(t *testing.T) {
		// Execute create command without branch name
		err := cliHelper.ExecuteCommand(rootCmd, "create")

		// Should fail
		require.Error(t, err)
		// Cobra should handle this as "accepts 1 arg(s), received 0"
	})
}

func TestCreateCommandIntegration(t *testing.T) {
	// Create a test Git repository
	testRepo := helpers.NewTestGitRepository(t, "integration-test")

	// Create some test files that might be auto-copied
	testRepo.CreateFile(".cursorrules", "# Cursor rules")
	testRepo.CreateFile("CLAUDE.md", "# Claude context")
	testRepo.CreateDirectory(".ai")
	testRepo.CreateFile(".ai/prompts.md", "# AI prompts")
	testRepo.CommitAll("Add test files")

	// Create CLI test helper
	cliHelper := helpers.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := helpers.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("full workflow", func(t *testing.T) {
		// Execute create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/integration-test")

		// Should succeed
		require.NoError(t, err)

		// Check output
		stdout := cliHelper.GetStdout()
		assert.Contains(t, stdout, "✅")
		assert.Contains(t, stdout, "integration-test-feature-integration-test")

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "integration-test-feature-integration-test")
		assert.DirExists(t, expectedPath)

		// Verify it's a proper Git worktree
		gitDir := filepath.Join(expectedPath, ".git")
		assert.FileExists(t, gitDir) // Should be a file pointing to the main .git
	})
}
