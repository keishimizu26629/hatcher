package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")

	// Change to the repository directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	// Test NewRepository
	repo, err := NewRepository()
	require.NoError(t, err)
	assert.NotNil(t, repo)

	// Test GetRoot
	root, err := repo.GetRoot()
	require.NoError(t, err)
	assert.Equal(t, testRepo.RepoDir, root)

	// Test GetProjectName
	assert.Equal(t, "test-project", repo.GetProjectName())

	// Test IsGitRepository
	assert.True(t, repo.IsGitRepository())
}

func TestNewRepositoryFromPath(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")

	// Test NewRepositoryFromPath
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)
	assert.NotNil(t, repo)

	// Verify repository information
	root, err := repo.GetRoot()
	require.NoError(t, err)
	assert.Equal(t, testRepo.RepoDir, root)
	assert.Equal(t, "test-project", repo.GetProjectName())
}

func TestNewRepository_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that's not a Git repository
	tempDir := t.TempDir()

	// Change to the non-Git directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test NewRepository should fail
	repo, err := NewRepository()
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestBranchExists(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Test existing branch (main/master should exist)
	currentBranch, err := repo.GetCurrentBranch()
	require.NoError(t, err)

	exists, err := repo.BranchExists(currentBranch)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing branch
	exists, err = repo.BranchExists("non-existing-branch")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateBranch(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create a new branch
	branchName := "feature/test-branch"
	err = repo.CreateBranch(branchName)
	require.NoError(t, err)

	// Verify the branch exists
	exists, err := repo.BranchExists(branchName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify we're on the new branch
	currentBranch, err := repo.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, branchName, currentBranch)
}

func TestDeleteBranch(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create a new branch
	branchName := "feature/to-delete"
	err = repo.CreateBranch(branchName)
	require.NoError(t, err)

	// Switch back to main branch
	mainBranch, err := repo.GetCurrentBranch()
	require.NoError(t, err)
	if mainBranch == branchName {
		// We need to switch to a different branch to delete the current one
		// Go back to the original branch (likely main or master)
		testRepo.SwitchToBranch("main")
	}

	// Delete the branch
	err = repo.DeleteBranch(branchName, false)
	require.NoError(t, err)

	// Verify the branch no longer exists
	exists, err := repo.BranchExists(branchName)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateWorktree(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create a worktree for a new branch
	branchName := "feature/test-worktree"
	worktreePath := filepath.Join(testRepo.TempDir, "test-project-feature-test-worktree")

	err = repo.CreateWorktree(worktreePath, branchName, true)
	require.NoError(t, err)

	// Verify the worktree directory exists
	assert.DirExists(t, worktreePath)

	// Verify the branch was created
	exists, err := repo.BranchExists(branchName)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRemoveWorktree(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create a worktree
	branchName := "feature/to-remove"
	worktreePath := filepath.Join(testRepo.TempDir, "test-project-feature-to-remove")

	err = repo.CreateWorktree(worktreePath, branchName, true)
	require.NoError(t, err)
	assert.DirExists(t, worktreePath)

	// Remove the worktree
	err = repo.RemoveWorktree(worktreePath, false)
	require.NoError(t, err)

	// Verify the worktree directory no longer exists
	assert.NoDirExists(t, worktreePath)
}

func TestListWorktrees(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// List worktrees (should have at least the main worktree)
	worktrees, err := repo.ListWorktrees()
	require.NoError(t, err)
	assert.NotEmpty(t, worktrees)

	// The main repository should be in the list
	found := false
	for _, wt := range worktrees {
		if wt.Path == testRepo.RepoDir {
			found = true
			break
		}
	}
	assert.True(t, found, "Main repository should be in worktree list")

	// Create an additional worktree
	branchName := "feature/list-test"
	worktreePath := filepath.Join(testRepo.TempDir, "test-project-feature-list-test")

	err = repo.CreateWorktree(worktreePath, branchName, true)
	require.NoError(t, err)

	// List worktrees again
	worktrees, err = repo.ListWorktrees()
	require.NoError(t, err)
	assert.Len(t, worktrees, 2, "Should have 2 worktrees after creating one")

	// Verify the new worktree is in the list
	found = false
	for _, wt := range worktrees {
		if wt.Path == worktreePath && wt.Branch == branchName {
			found = true
			break
		}
	}
	assert.True(t, found, "New worktree should be in the list")
}

func TestUpdateGitignore(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Test updating .gitignore with new files
	filesToIgnore := []string{".ai/", ".cursorrules", "CLAUDE.md"}
	err = repo.UpdateGitignore(filesToIgnore)
	require.NoError(t, err)

	// Verify .gitignore was created and contains the files
	gitignorePath := filepath.Join(testRepo.RepoDir, ".gitignore")
	assert.FileExists(t, gitignorePath)

	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	gitignoreContent := string(content)
	for _, file := range filesToIgnore {
		assert.Contains(t, gitignoreContent, file)
	}
	assert.Contains(t, gitignoreContent, "# Auto-copied files (added by hatcher)")
}

func TestGetWorktreePath(t *testing.T) {
	// Create a test Git repository
	testRepo := testutil.NewTestGitRepository(t, "test-project")
	repo, err := NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create a worktree
	branchName := "feature/path-test"
	worktreePath := filepath.Join(testRepo.TempDir, "test-project-feature-path-test")

	err = repo.CreateWorktree(worktreePath, branchName, true)
	require.NoError(t, err)

	// Test GetWorktreePath
	foundPath, err := repo.GetWorktreePath(branchName)
	require.NoError(t, err)
	assert.Equal(t, worktreePath, foundPath)

	// Test with non-existing branch
	_, err = repo.GetWorktreePath("non-existing-branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
