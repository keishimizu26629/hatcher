package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemover_RemoveWorktree(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "remover-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	remover := NewRemover(repo)

	t.Run("remove worktree only", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/remove-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remover-test-feature-remove-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)
		assert.DirExists(t, worktreePath)

		// Verify branch exists
		exists, err := repo.BranchExists(branchName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Remove worktree only
		options := RemoveOptions{
			BranchName:    branchName,
			RemoveBranch:  false,
			RemoveRemote:  false,
			Force:         false,
			SkipConfirm:   true, // Skip confirmation for tests
		}

		result, err := remover.RemoveWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, branchName, result.BranchName)
		assert.Equal(t, worktreePath, result.WorktreePath)
		assert.True(t, result.WorktreeRemoved)
		assert.False(t, result.LocalBranchRemoved)
		assert.False(t, result.RemoteBranchRemoved)

		// Verify worktree directory was removed
		assert.NoDirExists(t, worktreePath)

		// Verify branch still exists
		exists, err = repo.BranchExists(branchName)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("remove worktree and local branch", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/remove-branch-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remover-test-feature-remove-branch-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Remove worktree and local branch
		options := RemoveOptions{
			BranchName:   branchName,
			RemoveBranch: true,
			RemoveRemote: false,
			Force:        false,
			SkipConfirm:  true,
		}

		result, err := remover.RemoveWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.True(t, result.WorktreeRemoved)
		assert.True(t, result.LocalBranchRemoved)
		assert.False(t, result.RemoteBranchRemoved)

		// Verify worktree directory was removed
		assert.NoDirExists(t, worktreePath)

		// Verify branch was removed
		exists, err := repo.BranchExists(branchName)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("remove worktree with force when locked", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/force-remove-test"
		worktreePath := filepath.Join(testRepo.TempDir, "remover-test-feature-force-remove-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Create a file to potentially lock the worktree
		testFile := filepath.Join(worktreePath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Remove with force
		options := RemoveOptions{
			BranchName:   branchName,
			RemoveBranch: false,
			RemoveRemote: false,
			Force:        true,
			SkipConfirm:  true,
		}

		result, err := remover.RemoveWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify removal succeeded
		assert.True(t, result.WorktreeRemoved)
		assert.NoDirExists(t, worktreePath)
	})

	t.Run("remove non-existent worktree", func(t *testing.T) {
		// Try to remove non-existent worktree
		options := RemoveOptions{
			BranchName:   "feature/non-existent",
			RemoveBranch: false,
			RemoveRemote: false,
			Force:        false,
			SkipConfirm:  true,
		}

		result, err := remover.RemoveWorktree(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "worktree not found")
	})

	t.Run("remove main repository worktree", func(t *testing.T) {
		// Get current branch (main repository)
		currentBranch, err := repo.GetCurrentBranch()
		require.NoError(t, err)

		// Try to remove main repository
		options := RemoveOptions{
			BranchName:   currentBranch,
			RemoveBranch: false,
			RemoveRemote: false,
			Force:        false,
			SkipConfirm:  true,
		}

		result, err := remover.RemoveWorktree(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot remove main repository")
	})
}

func TestRemover_ValidateRemoval(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "validate-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	remover := NewRemover(repo)

	t.Run("validate safe removal", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/validate-test"
		worktreePath := filepath.Join(testRepo.TempDir, "validate-test-feature-validate-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Validate removal
		validation, err := remover.ValidateRemoval(branchName)
		require.NoError(t, err)
		assert.NotNil(t, validation)

		// Verify validation result
		assert.Equal(t, branchName, validation.BranchName)
		assert.Equal(t, worktreePath, validation.WorktreePath)
		assert.True(t, validation.WorktreeExists)
		assert.True(t, validation.LocalBranchExists)
		assert.False(t, validation.IsMainRepository)
		assert.True(t, validation.CanRemove)
		assert.Empty(t, validation.Warnings)
	})

	t.Run("validate removal of main repository", func(t *testing.T) {
		// Get current branch (main repository)
		currentBranch, err := repo.GetCurrentBranch()
		require.NoError(t, err)

		// Validate removal of main repository
		validation, err := remover.ValidateRemoval(currentBranch)
		require.NoError(t, err)
		assert.NotNil(t, validation)

		// Should detect main repository
		assert.True(t, validation.IsMainRepository)
		assert.False(t, validation.CanRemove)
		assert.Contains(t, validation.Warnings, "main repository")
	})

	t.Run("validate removal with uncommitted changes", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/uncommitted-test"
		worktreePath := filepath.Join(testRepo.TempDir, "validate-test-feature-uncommitted-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Create uncommitted changes
		testFile := filepath.Join(worktreePath, "uncommitted.txt")
		err = os.WriteFile(testFile, []byte("uncommitted content"), 0644)
		require.NoError(t, err)

		// Validate removal
		validation, err := remover.ValidateRemoval(branchName)
		require.NoError(t, err)
		assert.NotNil(t, validation)

		// Should warn about uncommitted changes
		assert.True(t, validation.CanRemove) // Can still remove with force
		assert.NotEmpty(t, validation.Warnings)
	})

	t.Run("validate removal of non-existent worktree", func(t *testing.T) {
		// Validate removal of non-existent worktree
		validation, err := remover.ValidateRemoval("feature/non-existent")
		require.NoError(t, err)
		assert.NotNil(t, validation)

		// Should indicate worktree doesn't exist
		assert.False(t, validation.WorktreeExists)
		assert.False(t, validation.CanRemove)
	})
}

func TestRemover_GetRemovalPlan(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "plan-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	remover := NewRemover(repo)

	t.Run("get removal plan for worktree only", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/plan-test"
		worktreePath := filepath.Join(testRepo.TempDir, "plan-test-feature-plan-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Get removal plan
		options := RemoveOptions{
			BranchName:   branchName,
			RemoveBranch: false,
			RemoveRemote: false,
			Force:        false,
			SkipConfirm:  false,
		}

		plan, err := remover.GetRemovalPlan(options)
		require.NoError(t, err)
		assert.NotNil(t, plan)

		// Verify plan
		assert.Equal(t, branchName, plan.BranchName)
		assert.Equal(t, worktreePath, plan.WorktreePath)
		assert.True(t, plan.WillRemoveWorktree)
		assert.False(t, plan.WillRemoveLocalBranch)
		assert.False(t, plan.WillRemoveRemoteBranch)
		assert.NotEmpty(t, plan.Description)
	})

	t.Run("get removal plan for everything", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/full-removal-test"
		worktreePath := filepath.Join(testRepo.TempDir, "plan-test-feature-full-removal-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Get removal plan for everything
		options := RemoveOptions{
			BranchName:   branchName,
			RemoveBranch: true,
			RemoveRemote: true,
			Force:        false,
			SkipConfirm:  false,
		}

		plan, err := remover.GetRemovalPlan(options)
		require.NoError(t, err)
		assert.NotNil(t, plan)

		// Verify plan
		assert.True(t, plan.WillRemoveWorktree)
		assert.True(t, plan.WillRemoveLocalBranch)
		assert.True(t, plan.WillRemoveRemoteBranch)
		assert.Contains(t, plan.Description, "worktree")
		assert.Contains(t, plan.Description, "local branch")
		assert.Contains(t, plan.Description, "remote branch")
	})
}

func TestRemover_ConfirmRemoval(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "confirm-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	remover := NewRemover(repo)

	t.Run("skip confirmation when requested", func(t *testing.T) {
		plan := &RemovalPlan{
			BranchName:             "feature/test",
			WorktreePath:           "/path/to/worktree",
			WillRemoveWorktree:     true,
			WillRemoveLocalBranch:  false,
			WillRemoveRemoteBranch: false,
			Description:            "Remove worktree only",
		}

		// Should return true when skip confirmation is enabled
		confirmed := remover.ConfirmRemoval(plan, true)
		assert.True(t, confirmed)
	})

	t.Run("require confirmation for dangerous operations", func(t *testing.T) {
		plan := &RemovalPlan{
			BranchName:             "main",
			WorktreePath:           "/path/to/main",
			WillRemoveWorktree:     true,
			WillRemoveLocalBranch:  true,
			WillRemoveRemoteBranch: true,
			Description:            "Remove everything",
			Warnings:               []string{"This will remove the main branch"},
		}

		// Should require confirmation for dangerous operations
		// In a real implementation, this would prompt the user
		// For testing, we'll simulate user declining
		confirmed := remover.ConfirmRemoval(plan, false)
		assert.False(t, confirmed) // Simulated user decline
	})
}
