package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveCommand_Integration(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "remove-integration-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("remove worktree only", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/remove-integration-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-remove-integration-test")

		// Create worktree using git command
		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)
		assert.True(t, testRepo.BranchExists(branchName))

		// Execute remove command
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--yes"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Successfully processed")
		assert.Contains(t, output, "Removed worktree")
		assert.NotContains(t, output, "Removed local branch")
		assert.NotContains(t, output, "Removed remote branch")

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)

		// Verify branch still exists
		assert.True(t, testRepo.BranchExists(branchName))
	})

	t.Run("remove worktree and local branch", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/remove-branch-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-remove-branch-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Execute remove command with --branch flag
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--branch", "--yes"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Successfully processed")
		assert.Contains(t, output, "Removed worktree")
		assert.Contains(t, output, "Removed local branch")
		assert.NotContains(t, output, "Removed remote branch")

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)

		// Verify branch was removed
		assert.False(t, testRepo.BranchExists(branchName))
	})

	t.Run("remove everything with --all flag", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/remove-all-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-remove-all-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Execute remove command with --all flag
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--all", "--yes"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Successfully processed")
		assert.Contains(t, output, "Removed worktree")
		assert.Contains(t, output, "Removed local branch")
		// Note: Remote branch removal might not show if no remote exists

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)

		// Verify branch was removed
		assert.False(t, testRepo.BranchExists(branchName))
	})

	t.Run("dry run mode", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/dry-run-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-dry-run-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Execute remove command in dry run mode
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--branch", "--dry-run"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Dry run mode")
		assert.Contains(t, output, "would perform the following actions")
		assert.Contains(t, output, branchName)

		// Verify nothing was actually removed
		assert.DirExists(t, worktreePath)
		assert.True(t, testRepo.BranchExists(branchName))
	})

	t.Run("remove non-existent worktree", func(t *testing.T) {
		// Try to remove non-existent worktree
		output, err := testutil.ExecuteCommand(removeCmd, []string{"feature/non-existent", "--yes"})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(output), "not found")
	})

	t.Run("remove main repository worktree", func(t *testing.T) {
		// Get current branch (main repository)
		currentBranch := strings.TrimSpace(testRepo.GetCurrentBranch())

		// Try to remove main repository
		output, err := testutil.ExecuteCommand(removeCmd, []string{currentBranch, "--yes"})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(output), "main repository")
	})

	t.Run("force removal", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/force-remove-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-force-remove-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Create a file to potentially lock the worktree
		testFile := filepath.Join(worktreePath, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Execute remove command with --force flag
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--force", "--yes"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Successfully processed")
		assert.Contains(t, output, "Removed worktree")

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)
	})

	t.Run("command aliases", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/alias-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-alias-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Test 'rm' alias
		output, err := testutil.ExecuteCommandByName("rm", []string{branchName, "--yes"})
		require.NoError(t, err)

		// Verify output
		assert.Contains(t, output, "Successfully processed")
		assert.Contains(t, output, "Removed worktree")

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)
	})

	t.Run("verbose output", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/verbose-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-integration-test-feature-verbose-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Execute remove command with verbose flag
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--yes", "--verbose"})
		require.NoError(t, err)

		// Verify verbose output
		assert.Contains(t, output, "Successfully processed")
		// Note: Specific verbose messages depend on implementation

		// Verify worktree was removed
		assert.NoDirExists(t, worktreePath)
	})
}

func TestRemoveCommand_Validation(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "remove-validation-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("missing branch name argument", func(t *testing.T) {
		// Execute remove command without branch name
		output, err := testutil.ExecuteCommand(removeCmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(output), "required")
	})

	t.Run("too many arguments", func(t *testing.T) {
		// Execute remove command with too many arguments
		output, err := testutil.ExecuteCommand(removeCmd, []string{"branch1", "branch2"})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(output), "accepts")
	})

	t.Run("invalid flag combinations", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/flag-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remove-validation-test-feature-flag-test")

		testRepo.CreateWorktree(worktreePath, branchName)
		assert.DirExists(t, worktreePath)

		// Test that --all implies --branch
		output, err := testutil.ExecuteCommand(removeCmd, []string{branchName, "--all", "--dry-run"})
		require.NoError(t, err)

		// Should show that both worktree and local branch will be removed
		assert.Contains(t, output, "local branch")
		assert.Contains(t, output, "remote branch")
	})
}
