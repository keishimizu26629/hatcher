package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGitRepository represents a test Git repository
type TestGitRepository struct {
	TempDir     string
	RepoDir     string
	ProjectName string
	t           *testing.T
}

// NewTestGitRepository creates a new test Git repository
func NewTestGitRepository(t *testing.T, projectName string) *TestGitRepository {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, projectName)

	repo := &TestGitRepository{
		TempDir:     tempDir,
		RepoDir:     repoDir,
		ProjectName: projectName,
		t:           t,
	}

	repo.initializeRepo()
	return repo
}

// initializeRepo initializes a Git repository with basic setup
func (r *TestGitRepository) initializeRepo() {
	// Create project directory
	require.NoError(r.t, os.MkdirAll(r.RepoDir, 0755))

	// Initialize Git repository
	r.runGitCommand("init")

	// Configure Git user (required for commits)
	r.runGitCommand("config", "user.name", "Test User")
	r.runGitCommand("config", "user.email", "test@example.com")

	// Create initial commit
	readmeFile := filepath.Join(r.RepoDir, "README.md")
	require.NoError(r.t, os.WriteFile(readmeFile, []byte("# Test Project"), 0644))

	r.runGitCommand("add", "README.md")
	r.runGitCommand("commit", "-m", "Initial commit")
}

// CreateBranch creates a new branch
func (r *TestGitRepository) CreateBranch(branchName string) {
	r.runGitCommand("checkout", "-b", branchName)
}

// CreateFile creates a file in the repository
func (r *TestGitRepository) CreateFile(relativePath, content string) {
	fullPath := filepath.Join(r.RepoDir, relativePath)
	dir := filepath.Dir(fullPath)
	require.NoError(r.t, os.MkdirAll(dir, 0755))
	require.NoError(r.t, os.WriteFile(fullPath, []byte(content), 0644))
}

// CreateDirectory creates a directory in the repository
func (r *TestGitRepository) CreateDirectory(relativePath string) {
	fullPath := filepath.Join(r.RepoDir, relativePath)
	require.NoError(r.t, os.MkdirAll(fullPath, 0755))
}

// CommitAll commits all changes
func (r *TestGitRepository) CommitAll(message string) {
	r.runGitCommand("add", ".")
	r.runGitCommand("commit", "-m", message)
}

// SwitchToBranch switches to an existing branch
func (r *TestGitRepository) SwitchToBranch(branchName string) {
	r.runGitCommand("checkout", branchName)
}

// GetCurrentBranch returns the current branch name
func (r *TestGitRepository) GetCurrentBranch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = r.RepoDir
	output, err := cmd.Output()
	require.NoError(r.t, err)
	return strings.TrimSpace(string(output))
}

// BranchExists checks if a branch exists
func (r *TestGitRepository) BranchExists(branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = r.RepoDir
	err := cmd.Run()
	return err == nil
}

// WorktreeExists checks if a worktree exists at the given path
func (r *TestGitRepository) WorktreeExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ListWorktrees returns a list of all worktrees
func (r *TestGitRepository) ListWorktrees() []string {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.RepoDir
	output, err := cmd.Output()
	require.NoError(r.t, err)

	// Parse worktree list output
	// This is a simplified parser for test purposes
	return []string{string(output)}
}

// runGitCommand runs a Git command in the repository directory
func (r *TestGitRepository) runGitCommand(args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.RepoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("Git command failed: git %v\nOutput: %s\nError: %v", args, output, err)
	}
}
