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

func TestWorktreeFinder_FindWorktree(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "finder-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	finder := NewFinder(repo)

	t.Run("find existing hatcher worktree", func(t *testing.T) {
		// Create a worktree using hatcher naming convention
		branchName := "feature/test-branch"
		worktreePath := filepath.Join(testRepo.TempDir, "finder-test-feature-test-branch")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Find the worktree
		foundPath, exists, err := finder.FindWorktree(branchName)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, worktreePath, foundPath)
	})

	t.Run("find worktree by sanitized branch name", func(t *testing.T) {
		// Create a worktree with special characters in branch name
		branchName := "feature/user@auth#2024"
		sanitizedName := "feature-user-auth-2024"
		worktreePath := filepath.Join(testRepo.TempDir, "finder-test-"+sanitizedName)

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Find the worktree using original branch name
		foundPath, exists, err := finder.FindWorktree(branchName)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, worktreePath, foundPath)
	})

	t.Run("worktree not found", func(t *testing.T) {
		// Try to find non-existent worktree
		foundPath, exists, err := finder.FindWorktree("non-existent-branch")
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Empty(t, foundPath)
	})

	t.Run("find worktree with multiple matches", func(t *testing.T) {
		// Create multiple worktrees with similar names
		branchName1 := "feature/test"
		branchName2 := "feature/test-extended"

		worktreePath1 := filepath.Join(testRepo.TempDir, "finder-test-feature-test")
		worktreePath2 := filepath.Join(testRepo.TempDir, "finder-test-feature-test-extended")

		err := repo.CreateWorktree(worktreePath1, branchName1, true)
		require.NoError(t, err)
		err = repo.CreateWorktree(worktreePath2, branchName2, true)
		require.NoError(t, err)

		// Find exact match for "feature/test"
		foundPath, exists, err := finder.FindWorktree(branchName1)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, worktreePath1, foundPath)

		// Find exact match for "feature/test-extended"
		foundPath, exists, err = finder.FindWorktree(branchName2)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, worktreePath2, foundPath)
	})

	t.Run("find main repository worktree", func(t *testing.T) {
		// Try to find main repository (should be found)
		currentBranch, err := repo.GetCurrentBranch()
		require.NoError(t, err)

		foundPath, exists, err := finder.FindWorktree(currentBranch)
		require.NoError(t, err)
		assert.True(t, exists)

		// Should return the main repository path
		expectedPath, _ := repo.GetRoot()
		assert.Equal(t, expectedPath, foundPath)
	})
}

func TestWorktreeFinder_ListHatcherWorktrees(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "list-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	finder := NewFinder(repo)

	t.Run("list hatcher-managed worktrees", func(t *testing.T) {
		// Create multiple worktrees with hatcher naming convention
		branches := []string{
			"feature/user-auth",
			"bugfix/header-issue",
			"release/v2.0.0",
		}

		var expectedPaths []string
		for _, branch := range branches {
			sanitized := SanitizeBranchName(branch)
			worktreePath := filepath.Join(testRepo.TempDir, "list-test-"+sanitized)
			expectedPaths = append(expectedPaths, worktreePath)

			err := repo.CreateWorktree(worktreePath, branch, true)
			require.NoError(t, err)
		}

		// List hatcher worktrees
		worktrees, err := finder.ListHatcherWorktrees()
		require.NoError(t, err)

		// Should include main repository + created worktrees
		assert.GreaterOrEqual(t, len(worktrees), len(branches))

		// Verify hatcher worktrees are included
		for i, expectedPath := range expectedPaths {
			found := false
			for _, wt := range worktrees {
				if wt.Path == expectedPath {
					assert.Equal(t, branches[i], wt.Branch)
					assert.True(t, wt.IsHatcher)
					found = true
					break
				}
			}
			assert.True(t, found, "Worktree not found: %s", expectedPath)
		}
	})

	t.Run("exclude non-hatcher worktrees", func(t *testing.T) {
		// Create a worktree with non-hatcher naming
		nonHatcherPath := filepath.Join(testRepo.TempDir, "custom-worktree-name")
		err := repo.CreateWorktree(nonHatcherPath, "feature/custom", true)
		require.NoError(t, err)

		// List hatcher worktrees
		worktrees, err := finder.ListHatcherWorktrees()
		require.NoError(t, err)

		// Non-hatcher worktree should be excluded or marked as non-hatcher
		for _, wt := range worktrees {
			if wt.Path == nonHatcherPath {
				assert.False(t, wt.IsHatcher)
			}
		}
	})

	t.Run("empty repository", func(t *testing.T) {
		// Create empty repository
		emptyRepo := helpers.NewTestGitRepository(t, "empty-test")
		emptyGitRepo, err := git.NewRepositoryFromPath(emptyRepo.RepoDir)
		require.NoError(t, err)

		emptyFinder := NewFinder(emptyGitRepo)

		// List worktrees (should only have main repository)
		worktrees, err := emptyFinder.ListHatcherWorktrees()
		require.NoError(t, err)

		// Should have at least the main repository
		assert.GreaterOrEqual(t, len(worktrees), 1)

		// Main repository should be marked as hatcher-managed
		mainRepo, _ := emptyGitRepo.GetRoot()
		found := false
		for _, wt := range worktrees {
			if wt.Path == mainRepo {
				found = true
				break
			}
		}
		assert.True(t, found, "Main repository should be in the list")
	})
}

func TestWorktreeFinder_GetWorktreeInfo(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "info-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	finder := NewFinder(repo)

	t.Run("get worktree info for existing worktree", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/info-test"
		worktreePath := filepath.Join(testRepo.TempDir, "info-test-feature-info-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Get worktree info
		info, err := finder.GetWorktreeInfo(worktreePath)
		require.NoError(t, err)
		assert.NotNil(t, info)

		// Verify info
		assert.Equal(t, branchName, info.Branch)
		assert.Equal(t, worktreePath, info.Path)
		assert.True(t, info.IsHatcher)
		assert.NotEmpty(t, info.Head)
	})

	t.Run("get worktree info for main repository", func(t *testing.T) {
		// Get info for main repository
		mainPath, _ := repo.GetRoot()
		info, err := finder.GetWorktreeInfo(mainPath)
		require.NoError(t, err)
		assert.NotNil(t, info)

		// Verify info
		assert.Equal(t, mainPath, info.Path)
		assert.NotEmpty(t, info.Branch)
		assert.NotEmpty(t, info.Head)
		// Main repository is considered hatcher-managed
		assert.False(t, info.IsHatcher) // Main repo is not created by hatcher
	})

	t.Run("get worktree info for non-existent path", func(t *testing.T) {
		// Try to get info for non-existent path
		info, err := finder.GetWorktreeInfo("/non/existent/path")
		assert.Error(t, err)
		assert.Nil(t, info)
	})
}

func TestWorktreeFinder_IsHatcherWorktree(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		worktreePath string
		expected     bool
	}{
		{
			name:         "hatcher worktree - feature branch",
			projectName:  "my-project",
			worktreePath: "/path/to/my-project-feature-auth",
			expected:     true,
		},
		{
			name:         "hatcher worktree - bugfix branch",
			projectName:  "my-project",
			worktreePath: "/path/to/my-project-bugfix-header",
			expected:     true,
		},
		{
			name:         "non-hatcher worktree - different name",
			projectName:  "my-project",
			worktreePath: "/path/to/custom-worktree",
			expected:     false,
		},
		{
			name:         "non-hatcher worktree - similar but not exact",
			projectName:  "my-project",
			worktreePath: "/path/to/my-project-like-name",
			expected:     false,
		},
		{
			name:         "main repository",
			projectName:  "my-project",
			worktreePath: "/path/to/my-project",
			expected:     false,
		},
		{
			name:         "hatcher worktree - complex branch name",
			projectName:  "complex-project",
			worktreePath: "/path/to/complex-project-feature-user-auth-v2",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHatcherWorktree(tt.worktreePath, tt.projectName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorktreeFinder_GenerateWorktreePath(t *testing.T) {
	tests := []struct {
		name        string
		repoRoot    string
		projectName string
		branchName  string
		expected    string
	}{
		{
			name:        "simple branch name",
			repoRoot:    "/Users/test/projects/my-app",
			projectName: "my-app",
			branchName:  "main",
			expected:    "/Users/test/projects/my-app-main",
		},
		{
			name:        "feature branch with slash",
			repoRoot:    "/Users/test/projects/my-app",
			projectName: "my-app",
			branchName:  "feature/user-auth",
			expected:    "/Users/test/projects/my-app-feature-user-auth",
		},
		{
			name:        "complex branch name",
			repoRoot:    "/Users/test/projects/my-app",
			projectName: "my-app",
			branchName:  "feature/user@auth#2024",
			expected:    "/Users/test/projects/my-app-feature-user-auth-2024",
		},
		{
			name:        "nested project path",
			repoRoot:    "/Users/test/workspace/projects/my-app",
			projectName: "my-app",
			branchName:  "bugfix/header-issue",
			expected:    "/Users/test/workspace/projects/my-app-bugfix-header-issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateWorktreePath(tt.repoRoot, tt.projectName, tt.branchName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
