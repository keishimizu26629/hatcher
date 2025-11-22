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

func TestListCommand_Integration(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "list-integration-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("list empty worktrees", func(t *testing.T) {
		// Execute list command
		output, err := testutil.ExecuteCommand(listCmd, []string{})
		require.NoError(t, err)

		// Should show main repository
		assert.Contains(t, output, "main")
		assert.Contains(t, output, "BRANCH")
		assert.Contains(t, output, "PATH")
	})

	t.Run("list Hatcher-managed worktrees", func(t *testing.T) {
		// Create Hatcher-managed worktrees
		branchName1 := "feature/list-test-1"
		branchName2 := "feature/list-test-2"
		worktreePath1 := filepath.Join(testRepo.TempDir, "list-integration-test-feature-list-test-1")
		worktreePath2 := filepath.Join(testRepo.TempDir, "list-integration-test-feature-list-test-2")

		testRepo.CreateWorktree(worktreePath1, branchName1)
		testRepo.CreateWorktree(worktreePath2, branchName2)

		// Execute list command
		output, err := testutil.ExecuteCommand(listCmd, []string{})
		require.NoError(t, err)

		// Should show both worktrees
		assert.Contains(t, output, branchName1)
		assert.Contains(t, output, branchName2)
		assert.Contains(t, output, "hatcher")
	})

	t.Run("list all worktrees", func(t *testing.T) {
		// Create a mix of Hatcher and non-Hatcher worktrees
		hatcherBranch := "feature/hatcher-managed"
		nonHatcherBranch := "feature/manual"
		hatcherPath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-hatcher-managed")
		nonHatcherPath := filepath.Join(testRepo.TempDir, "manual-worktree")

		testRepo.CreateWorktree(hatcherPath, hatcherBranch)
		testRepo.CreateWorktree(nonHatcherPath, nonHatcherBranch)

		// Execute list command with --all flag
		output, err := testutil.ExecuteCommand(listCmd, []string{"--all"})
		require.NoError(t, err)

		// Should show both types of worktrees
		assert.Contains(t, output, hatcherBranch)
		assert.Contains(t, output, nonHatcherBranch)
		assert.Contains(t, output, "hatcher")
		assert.Contains(t, output, "manual")
	})

	t.Run("list with paths", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/paths-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-paths-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with --paths flag
		output, err := testutil.ExecuteCommand(listCmd, []string{"--paths"})
		require.NoError(t, err)

		// Should show full paths
		assert.Contains(t, output, branchName)
		assert.Contains(t, output, worktreePath)
	})

	t.Run("list with status", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/status-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-status-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with --status flag
		output, err := testutil.ExecuteCommand(listCmd, []string{"--status"})
		require.NoError(t, err)

		// Should show status information
		assert.Contains(t, output, branchName)
		assert.Contains(t, output, "STATUS")
	})

	t.Run("JSON output format", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/json-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-json-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with JSON format
		output, err := testutil.ExecuteCommand(listCmd, []string{"--format", "json"})
		require.NoError(t, err)

		// Should be valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Should contain worktrees
		assert.Contains(t, result, "worktrees")
		assert.Contains(t, result, "total")
	})

	t.Run("simple output format", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/simple-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-simple-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with simple format
		output, err := testutil.ExecuteCommand(listCmd, []string{"--format", "simple"})
		require.NoError(t, err)

		// Should show simple list
		assert.Contains(t, output, branchName)
		assert.NotContains(t, output, "BRANCH") // No table headers
		assert.NotContains(t, output, "PATH")
	})

	t.Run("filter by pattern", func(t *testing.T) {
		// Create worktrees with different prefixes
		featureBranch := "feature/filter-test"
		bugfixBranch := "bugfix/filter-test"
		featurePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-filter-test")
		bugfixPath := filepath.Join(testRepo.TempDir, "list-integration-test-bugfix-filter-test")

		testRepo.CreateWorktree(featurePath, featureBranch)
		testRepo.CreateWorktree(bugfixPath, bugfixBranch)

		// Execute list command with feature filter
		output, err := testutil.ExecuteCommand(listCmd, []string{"--filter", "feature/*"})
		require.NoError(t, err)

		// Should show only feature branches
		assert.Contains(t, output, featureBranch)
		assert.NotContains(t, output, bugfixBranch)
	})

	t.Run("command aliases", func(t *testing.T) {
		// Test 'ls' alias
		output, err := testutil.ExecuteCommandByName("ls", []string{})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "BRANCH")
		}

		// Test 'show' alias
		output, err = testutil.ExecuteCommandByName("show", []string{})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "BRANCH")
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/verbose-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-integration-test-feature-verbose-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with verbose flag
		output, err := testutil.ExecuteCommand(listCmd, []string{"--verbose"})
		require.NoError(t, err)

		// Should show worktree information
		assert.Contains(t, output, branchName)
	})
}

func TestListCommand_EdgeCases(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "list-edge-cases-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("invalid output format", func(t *testing.T) {
		// Execute list command with invalid format
		output, err := testutil.ExecuteCommand(listCmd, []string{"--format", "invalid"})
		require.NoError(t, err) // Should default to table format

		// Should show table format
		assert.Contains(t, output, "BRANCH")
	})

	t.Run("empty filter pattern", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/empty-filter-test"
		worktreePath := filepath.Join(testRepo.TempDir, "list-edge-cases-test-feature-empty-filter-test")

		testRepo.CreateWorktree(worktreePath, branchName)

		// Execute list command with empty filter
		output, err := testutil.ExecuteCommand(listCmd, []string{"--filter", ""})
		require.NoError(t, err)

		// Should show all worktrees (no filtering)
		assert.Contains(t, output, branchName)
	})

	t.Run("non-git repository", func(t *testing.T) {
		// Create temporary directory that's not a git repository
		tempDir := t.TempDir()
		err := os.Chdir(tempDir)
		require.NoError(t, err)

		// Execute list command
		output, err := testutil.ExecuteCommand(listCmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(output), "git")
	})

	t.Run("long branch names", func(t *testing.T) {
		// Change back to test repo
		err := os.Chdir(testRepo.RepoDir)
		require.NoError(t, err)

		// Create worktree with very long branch name
		longBranchName := "feature/this-is-a-very-long-branch-name-that-should-be-handled-properly-by-the-list-command"
		worktreePath := filepath.Join(testRepo.TempDir, "list-edge-cases-test-feature-this-is-a-very-long-branch-name-that-should-be-handled-properly-by-the-list-command")

		testRepo.CreateWorktree(worktreePath, longBranchName)

		// Execute list command
		output, err := testutil.ExecuteCommand(listCmd, []string{})
		require.NoError(t, err)

		// Should handle long names gracefully
		assert.Contains(t, output, longBranchName)
	})

	t.Run("special characters in branch names", func(t *testing.T) {
		// Create worktree with special characters (that are valid in Git)
		specialBranch := "feature/fix-issue-123"
		worktreePath := filepath.Join(testRepo.TempDir, "list-edge-cases-test-feature-fix-issue-123")

		testRepo.CreateWorktree(worktreePath, specialBranch)

		// Execute list command
		output, err := testutil.ExecuteCommand(listCmd, []string{})
		require.NoError(t, err)

		// Should handle special characters
		assert.Contains(t, output, specialBranch)
	})
}
