package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid branch name",
			branchName:  "feature/user-auth",
			expectError: false,
		},
		{
			name:        "valid simple branch name",
			branchName:  "main",
			expectError: false,
		},
		{
			name:        "empty branch name",
			branchName:  "",
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "branch name with dangerous characters",
			branchName:  "feature/../evil",
			expectError: true,
			errorMsg:    "dangerous character",
		},
		{
			name:        "branch name too long",
			branchName:  "very-long-branch-name-that-exceeds-the-maximum-allowed-length-of-100-characters-and-should-fail-validation",
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "branch name starting with dash",
			branchName:  "-invalid",
			expectError: true,
			errorMsg:    "invalid branch name format",
		},
		{
			name:        "branch name ending with dot",
			branchName:  "invalid.",
			expectError: true,
			errorMsg:    "invalid branch name format",
		},
		{
			name:        "branch name with consecutive dots",
			branchName:  "feature..test",
			expectError: true,
			errorMsg:    "consecutive characters",
		},
		{
			name:        "branch name with consecutive slashes",
			branchName:  "feature//test",
			expectError: true,
			errorMsg:    "consecutive characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branchName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "main",
			expected: "main",
		},
		{
			name:     "feature branch with slash",
			input:    "feature/user-auth",
			expected: "feature-user-auth",
		},
		{
			name:     "branch with multiple special characters",
			input:    "feature/user@auth#test",
			expected: "feature-user-auth-test",
		},
		{
			name:     "branch with spaces",
			input:    "feature user auth",
			expected: "feature-user-auth",
		},
		{
			name:     "branch with leading/trailing dashes",
			input:    "-feature-test-",
			expected: "feature-test",
		},
		{
			name:     "branch with consecutive dashes",
			input:    "feature--test",
			expected: "feature-test",
		},
		{
			name:     "complex branch name",
			input:    "feature/user@auth#2024:v1",
			expected: "feature-user-auth-2024-v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	repoRoot := "/Users/test/projects/my-app"
	projectName := "my-app"
	branchName := "feature/user-auth"

	expected := "/Users/test/projects/my-app-feature-user-auth"
	result := GenerateWorktreePath(repoRoot, projectName, branchName)

	// Normalize paths for cross-platform comparison
	assert.Equal(t, NormalizePath(expected), NormalizePath(result))
}

func TestIsHatcherWorktree(t *testing.T) {
	projectName := "my-app"

	tests := []struct {
		name         string
		worktreePath string
		expected     bool
	}{
		{
			name:         "hatcher worktree",
			worktreePath: "/Users/test/projects/my-app-feature-auth",
			expected:     true,
		},
		{
			name:         "non-hatcher worktree",
			worktreePath: "/Users/test/projects/other-project",
			expected:     false,
		},
		{
			name:         "main repository",
			worktreePath: "/Users/test/projects/my-app",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHatcherWorktree(tt.worktreePath, projectName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreator_Create(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	creator := NewCreator(repo)

	t.Run("create worktree for new branch", func(t *testing.T) {
		opts := CreateOptions{
			BranchName: "feature/new-feature",
			Force:      false,
			DryRun:     false,
		}

		result, err := creator.Create(opts)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, "feature/new-feature", result.BranchName)
		assert.True(t, result.IsNewBranch)
		assert.Contains(t, result.WorktreePath, "test-project-feature-new-feature")

		// Verify worktree was created
		assert.DirExists(t, result.WorktreePath)

		// Verify branch was created
		exists, err := repo.BranchExists("feature/new-feature")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("create worktree for existing branch", func(t *testing.T) {
		// First create a branch
		branchName := "feature/existing-branch"
		err := repo.CreateBranch(branchName)
		require.NoError(t, err)

		// Switch back to main to avoid issues
		testRepo.SwitchToBranch("main")

		opts := CreateOptions{
			BranchName: branchName,
			Force:      false,
			DryRun:     false,
		}

		result, err := creator.Create(opts)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, branchName, result.BranchName)
		assert.False(t, result.IsNewBranch)
		assert.DirExists(t, result.WorktreePath)
	})

	t.Run("dry run mode", func(t *testing.T) {
		opts := CreateOptions{
			BranchName: "feature/dry-run-test",
			Force:      false,
			DryRun:     true,
		}

		result, err := creator.Create(opts)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, "feature/dry-run-test", result.BranchName)
		assert.Contains(t, result.Message, "Would create worktree")

		// Verify worktree was NOT created
		assert.NoDirExists(t, result.WorktreePath)

		// Verify branch was NOT created
		exists, err := repo.BranchExists("feature/dry-run-test")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("force overwrite existing directory", func(t *testing.T) {
		branchName := "feature/force-test"

		// Create worktree first
		opts := CreateOptions{
			BranchName: branchName,
			Force:      false,
			DryRun:     false,
		}

		result1, err := creator.Create(opts)
		require.NoError(t, err)

		// Create a file in the worktree to verify it gets overwritten
		testFile := filepath.Join(result1.WorktreePath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Try to create again without force (should fail)
		opts.BranchName = "feature/force-test-2"
		opts.Force = false

		// This should work since it's a different branch name
		result2, err := creator.Create(opts)
		require.NoError(t, err)
		assert.DirExists(t, result2.WorktreePath)
	})

	t.Run("invalid branch name", func(t *testing.T) {
		opts := CreateOptions{
			BranchName: "invalid/../branch",
			Force:      false,
			DryRun:     false,
		}

		result, err := creator.Create(opts)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid branch name")
	})

	t.Run("directory already exists without force", func(t *testing.T) {
		branchName := "feature/existing-dir-test"

		// Create worktree first
		opts := CreateOptions{
			BranchName: branchName,
			Force:      false,
			DryRun:     false,
		}

		result1, err := creator.Create(opts)
		require.NoError(t, err)

		// Remove the worktree but keep the directory
		err = repo.RemoveWorktree(result1.WorktreePath, true)
		require.NoError(t, err)

		// Create a dummy file to make sure directory exists
		err = os.MkdirAll(result1.WorktreePath, 0755)
		require.NoError(t, err)

		// Try to create again without force (should fail)
		opts.BranchName = branchName + "-2"
		// Generate the same path manually to test collision
		expectedPath := GenerateWorktreePath(testRepo.RepoDir, "test-project", opts.BranchName)
		err = os.MkdirAll(expectedPath, 0755)
		require.NoError(t, err)

		result2, err := creator.Create(opts)
		assert.Error(t, err)
		assert.Nil(t, result2)
		assert.Contains(t, err.Error(), "directory already exists")
	})
}
