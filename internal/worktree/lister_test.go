package worktree

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLister_ListWorktrees(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "lister-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	lister := NewLister(repo)

	t.Run("list empty worktrees", func(t *testing.T) {
		// List worktrees when only main repository exists
		options := ListOptions{
			ShowAll:    false,
			ShowPaths:  false,
			ShowStatus: false,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have at least the main repository
		assert.GreaterOrEqual(t, len(result.Worktrees), 1)

		// Main repository should be marked as such
		mainFound := false
		for _, wt := range result.Worktrees {
			if wt.IsMain {
				mainFound = true
				assert.Equal(t, testRepo.RepoDir, wt.Path)
				break
			}
		}
		assert.True(t, mainFound, "Main repository should be found")
	})

	t.Run("list worktrees with Hatcher naming", func(t *testing.T) {
		// Create worktrees with Hatcher naming convention
		branchName1 := "feature/list-test-1"
		branchName2 := "feature/list-test-2"
		worktreePath1 := filepath.Join(testRepo.TempDir, "lister-test-feature-list-test-1")
		worktreePath2 := filepath.Join(testRepo.TempDir, "lister-test-feature-list-test-2")

		err := repo.CreateWorktree(worktreePath1, branchName1, true)
		require.NoError(t, err)
		err = repo.CreateWorktree(worktreePath2, branchName2, true)
		require.NoError(t, err)

		// List all worktrees
		options := ListOptions{
			ShowAll:    true,
			ShowPaths:  true,
			ShowStatus: false,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have main + 2 worktrees
		assert.GreaterOrEqual(t, len(result.Worktrees), 3)

		// Find our created worktrees
		var found1, found2 bool
		for _, wt := range result.Worktrees {
			if wt.Branch == branchName1 {
				found1 = true
				assert.Equal(t, worktreePath1, wt.Path)
				assert.False(t, wt.IsMain)
				assert.True(t, wt.IsHatcherManaged)
			}
			if wt.Branch == branchName2 {
				found2 = true
				assert.Equal(t, worktreePath2, wt.Path)
				assert.False(t, wt.IsMain)
				assert.True(t, wt.IsHatcherManaged)
			}
		}
		assert.True(t, found1, "First worktree should be found")
		assert.True(t, found2, "Second worktree should be found")
	})

	t.Run("list only Hatcher-managed worktrees", func(t *testing.T) {
		// Create a mix of Hatcher and non-Hatcher worktrees
		hatcherBranch := "feature/hatcher-managed"
		nonHatcherBranch := "feature/manual"
		hatcherPath := filepath.Join(testRepo.TempDir, "lister-test-feature-hatcher-managed")
		nonHatcherPath := filepath.Join(testRepo.TempDir, "manual-worktree")

		err := repo.CreateWorktree(hatcherPath, hatcherBranch, true)
		require.NoError(t, err)
		err = repo.CreateWorktree(nonHatcherPath, nonHatcherBranch, true)
		require.NoError(t, err)

		// List only Hatcher-managed worktrees
		options := ListOptions{
			ShowAll:    false, // Only Hatcher-managed
			ShowPaths:  true,
			ShowStatus: false,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Find worktrees
		var hatcherFound, nonHatcherFound bool
		for _, wt := range result.Worktrees {
			if wt.Branch == hatcherBranch {
				hatcherFound = true
				assert.True(t, wt.IsHatcherManaged)
			}
			if wt.Branch == nonHatcherBranch {
				nonHatcherFound = true
			}
		}

		assert.True(t, hatcherFound, "Hatcher-managed worktree should be found")
		assert.False(t, nonHatcherFound, "Non-Hatcher worktree should not be found when ShowAll=false")
	})

	t.Run("list with status information", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/status-test"
		worktreePath := filepath.Join(testRepo.TempDir, "lister-test-feature-status-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// List with status
		options := ListOptions{
			ShowAll:    true,
			ShowPaths:  true,
			ShowStatus: true,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Find our worktree and check status
		var found bool
		for _, wt := range result.Worktrees {
			if wt.Branch == branchName {
				found = true
				assert.NotEmpty(t, wt.Status) // Status should be populated
				break
			}
		}
		assert.True(t, found, "Worktree should be found")
	})

	t.Run("format output", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/format-test"
		worktreePath := filepath.Join(testRepo.TempDir, "lister-test-feature-format-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// List worktrees
		options := ListOptions{
			ShowAll:    true,
			ShowPaths:  true,
			ShowStatus: true,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)

		// Test different output formats
		tableOutput := result.FormatAsTable()
		assert.NotEmpty(t, tableOutput)
		assert.Contains(t, tableOutput, branchName)

		jsonOutput := result.FormatAsJSON()
		assert.NotEmpty(t, jsonOutput)
		assert.Contains(t, jsonOutput, branchName)

		simpleOutput := result.FormatAsSimple()
		assert.NotEmpty(t, simpleOutput)
		assert.Contains(t, simpleOutput, branchName)
	})
}

func TestLister_GetWorktreeStatus(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "status-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	lister := NewLister(repo)

	t.Run("get status of clean worktree", func(t *testing.T) {
		// Create a clean worktree
		branchName := "feature/clean-test"
		worktreePath := filepath.Join(testRepo.TempDir, "status-test-feature-clean-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Get status
		status, err := lister.GetWorktreeStatus(worktreePath)
		require.NoError(t, err)

		// Should be clean
		assert.Equal(t, StatusClean, status)
	})

	t.Run("get status of main repository", func(t *testing.T) {
		// Get status of main repository
		status, err := lister.GetWorktreeStatus(testRepo.RepoDir)
		require.NoError(t, err)

		// Should have some status
		assert.NotEmpty(t, status)
	})
}

func TestLister_FilterWorktrees(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "filter-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	lister := NewLister(repo)

	t.Run("filter by branch pattern", func(t *testing.T) {
		// Create worktrees with different branch names
		branches := []string{
			"feature/ui-improvements",
			"feature/api-changes",
			"bugfix/critical-fix",
			"hotfix/security-patch",
		}

		for _, branch := range branches {
			worktreePath := filepath.Join(testRepo.TempDir, "filter-test-"+branch[strings.LastIndex(branch, "/")+1:])
			err := repo.CreateWorktree(worktreePath, branch, true)
			require.NoError(t, err)
		}

		// List all worktrees first
		options := ListOptions{
			ShowAll:    true,
			ShowPaths:  false,
			ShowStatus: false,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)

		// Filter by pattern
		featureWorktrees := result.FilterByBranchPattern("feature/*")
		assert.GreaterOrEqual(t, len(featureWorktrees), 2)

		for _, wt := range featureWorktrees {
			assert.Contains(t, wt.Branch, "feature/")
		}

		bugfixWorktrees := result.FilterByBranchPattern("bugfix/*")
		assert.GreaterOrEqual(t, len(bugfixWorktrees), 1)

		for _, wt := range bugfixWorktrees {
			assert.Contains(t, wt.Branch, "bugfix/")
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		// List all worktrees with status
		options := ListOptions{
			ShowAll:    true,
			ShowPaths:  false,
			ShowStatus: true,
		}

		result, err := lister.ListWorktrees(options)
		require.NoError(t, err)

		// Filter by status
		cleanWorktrees := result.FilterByStatus(StatusClean)
		assert.GreaterOrEqual(t, len(cleanWorktrees), 0)

		for _, wt := range cleanWorktrees {
			assert.Equal(t, StatusClean, wt.Status)
		}
	})
}
