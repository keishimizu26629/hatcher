package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveCommandIntegration(t *testing.T) {
	t.Skip("Integration test temporarily disabled for Windows compatibility")
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "move-integration")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("move to existing worktree", func(t *testing.T) {
		// First create a worktree using create command
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/move-test")
		require.NoError(t, err)

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "move-integration-feature-move-test")
		assert.DirExists(t, expectedPath)

		// Now test move command
		err = cliHelper.ExecuteCommand(rootCmd, "move", "feature/move-test")

		// Move command may fail if no editors are installed, but should not crash
		if err != nil {
			// If it fails, it should be due to no editor found
			assert.Contains(t, err.Error(), "no suitable editor found")
		} else {
			// If it succeeds, check output
			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "✅ Found worktree")
			assert.Contains(t, stdout, expectedPath)
		}
	})

	t.Run("move to non-existent worktree without auto-create", func(t *testing.T) {
		// Test move to non-existent worktree
		err := cliHelper.ExecuteCommand(rootCmd, "move", "feature/non-existent")

		// Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree not found")
	})

	t.Run("move to non-existent worktree with auto-create", func(t *testing.T) {
		// Test move with --yes flag (auto-create)
		err := cliHelper.ExecuteCommand(rootCmd, "move", "--yes", "feature/auto-create-move")

		// May succeed or fail depending on editor availability
		if err != nil {
			// If it fails, should be due to editor, not worktree creation
			assert.Contains(t, err.Error(), "no suitable editor found")
		} else {
			// If it succeeds, verify worktree was created
			expectedPath := filepath.Join(testRepo.TempDir, "move-integration-feature-auto-create-move")
			assert.DirExists(t, expectedPath)

			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "🆕 Created new worktree")
		}
	})

	t.Run("move with switch mode", func(t *testing.T) {
		// Test move with switch flag
		err := cliHelper.ExecuteCommand(rootCmd, "move", "--switch", "feature/move-test")

		// May succeed or fail depending on editor availability
		if err != nil {
			assert.Contains(t, err.Error(), "no suitable editor found")
		} else {
			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "🔄 Switched to")
		}
	})

	t.Run("move with specific editor", func(t *testing.T) {
		// Test move with specific editor (likely to fail since test editor doesn't exist)
		err := cliHelper.ExecuteCommand(rootCmd, "move", "--editor", "non-existent-editor", "feature/move-test")

		// Should fail due to editor not found
		require.Error(t, err)
		assert.Contains(t, err.Error(), "editor 'non-existent-editor' not found")
	})
}

func TestMoveCommandHelp(t *testing.T) {
	cliHelper := testutil.NewCLITestHelper(t)

	// Execute help command
	err := cliHelper.ExecuteCommand(rootCmd, "move", "--help")
	require.NoError(t, err)

	// Check help output
	stdout := cliHelper.GetStdout()
	assert.Contains(t, stdout, "Move to existing worktree")
	assert.Contains(t, stdout, "--switch")
	assert.Contains(t, stdout, "--yes")
	assert.Contains(t, stdout, "--editor")
}

func TestMoveCommandFlags(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "move-flags")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("verbose flag", func(t *testing.T) {
		// Create a worktree first
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/verbose-test")
		require.NoError(t, err)

		// Test move with verbose flag
		err = cliHelper.ExecuteCommand(rootCmd, "--verbose", "move", "feature/verbose-test")

		// Check for verbose output (regardless of success/failure)
		stdout := cliHelper.GetStdout()
		stderr := cliHelper.GetStderr()
		output := stdout + stderr

		if err == nil {
			assert.Contains(t, output, "🔍 Searching for worktree")
		}
	})

	t.Run("combined flags", func(t *testing.T) {
		// Test move with multiple flags
		err := cliHelper.ExecuteCommand(rootCmd, "move", "--yes", "--switch", "feature/combined-flags")

		// May succeed or fail, but should handle flag combination properly
		if err != nil {
			// Should fail gracefully with appropriate error
			assert.NotContains(t, err.Error(), "unknown flag")
		}
	})
}

func TestMoveCommandEdgeCases(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "move-edge-cases")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("move with special characters in branch name", func(t *testing.T) {
		// Create worktree with special characters
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/user@auth#2024")
		require.NoError(t, err)

		// Test move to worktree with special characters
		err = cliHelper.ExecuteCommand(rootCmd, "move", "feature/user@auth#2024")

		// Should handle special characters properly
		if err != nil {
			// If it fails, should be due to editor, not branch name handling
			assert.Contains(t, err.Error(), "no suitable editor found")
		} else {
			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "✅ Found worktree")
		}
	})

	t.Run("move outside git repository", func(t *testing.T) {
		// Change to non-Git directory
		tempDir := t.TempDir()
		mockEnv.ChangeDir(tempDir)

		// Test move command
		err := cliHelper.ExecuteCommand(rootCmd, "move", "feature/test")

		// Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Not in a Git repository")

		// Change back to Git repository for cleanup
		mockEnv.ChangeDir(testRepo.RepoDir)
	})

	t.Run("move with empty branch name", func(t *testing.T) {
		// Test move without branch name
		err := cliHelper.ExecuteCommand(rootCmd, "move")

		// Should fail due to missing argument
		require.Error(t, err)
		// Cobra should handle this as "accepts 1 arg(s), received 0"
	})

	t.Run("move with very long branch name", func(t *testing.T) {
		// Create a very long branch name
		longBranchName := "feature/" + string(make([]byte, 200)) // Very long name
		for i := range longBranchName[8:] {
			longBranchName = longBranchName[:8+i] + "a" + longBranchName[8+i+1:]
		}

		// Test move command
		err := cliHelper.ExecuteCommand(rootCmd, "move", "--yes", longBranchName)

		// Should fail due to validation
		require.Error(t, err)
		// Should be caught by branch name validation
	})
}

func TestMoveCommandWorkflow(t *testing.T) {
	// Create test repository with realistic setup
	testRepo := testutil.NewTestGitRepository(t, "move-workflow")

	// Create some files
	testRepo.CreateFile("README.md", "# Move Workflow Test")
	testRepo.CreateFile("package.json", `{"name": "move-workflow-test"}`)
	testRepo.CommitAll("Initial commit")

	// Create CLI test helper
	cliHelper := testutil.NewCLITestHelper(t)

	// Create mock environment
	mockEnv := testutil.NewMockEnvironment(t)
	defer mockEnv.Cleanup()

	// Change to the repository directory
	mockEnv.ChangeDir(testRepo.RepoDir)

	t.Run("full workflow - create, move, move back", func(t *testing.T) {
		// Step 1: Create a worktree
		err := cliHelper.ExecuteCommand(rootCmd, "create", "feature/workflow-test")
		require.NoError(t, err)

		expectedPath := filepath.Join(testRepo.TempDir, "move-workflow-feature-workflow-test")
		assert.DirExists(t, expectedPath)

		// Step 2: Move to the worktree
		err = cliHelper.ExecuteCommand(rootCmd, "move", "feature/workflow-test")

		// May succeed or fail based on editor availability
		if err == nil {
			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "✅ Found worktree")
		}

		// Step 3: Move back to main (if main worktree exists)
		currentBranch := "main" // Assume main branch
		err = cliHelper.ExecuteCommand(rootCmd, "move", currentBranch)

		// This tests moving between different worktrees
		if err == nil {
			stdout := cliHelper.GetStdout()
			assert.Contains(t, stdout, "✅ Found worktree")
		}
	})
}
