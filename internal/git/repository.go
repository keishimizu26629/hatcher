package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository represents a Git repository
type Repository interface {
	// Repository information
	GetRoot() (string, error)
	GetProjectName() string
	IsGitRepository() bool

	// Branch operations
	BranchExists(branch string) (bool, error)
	RemoteBranchExists(branch string) (bool, error)
	GetCurrentBranch() (string, error)
	CreateBranch(branch string) error
	RemoveBranch(branch string, force bool) error
	RemoveRemoteBranch(branch string) error

	// Worktree operations
	CreateWorktree(path, branch string, newBranch bool) error
	RemoveWorktree(path string, force bool) error
	ListWorktrees() ([]Worktree, error)
	GetWorktreePath(branch string) (string, error)

	// Other operations
	UpdateGitignore(files []string) error
}

// Worktree represents a Git worktree
type Worktree struct {
	Branch string
	Path   string
	Head   string
	Status WorktreeStatus
}

// WorktreeStatus represents the status of a worktree
type WorktreeStatus string

const (
	StatusClean   WorktreeStatus = "clean"
	StatusDirty   WorktreeStatus = "dirty"
	StatusActive  WorktreeStatus = "active"
	StatusUnknown WorktreeStatus = "unknown"
)

// GitRepository implements the Repository interface
type GitRepository struct {
	root        string
	projectName string
}

// NewRepository creates a new Git repository instance
func NewRepository() (*GitRepository, error) {
	root, err := getGitRoot()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	projectName := filepath.Base(root)

	return &GitRepository{
		root:        root,
		projectName: projectName,
	}, nil
}

// NewRepositoryFromPath creates a new Git repository instance from a specific path
func NewRepositoryFromPath(path string) (*GitRepository, error) {
	// Change to the specified directory temporarily
	originalWd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if err := os.Chdir(path); err != nil {
		return nil, err
	}
	defer os.Chdir(originalWd)

	return NewRepository()
}

// GetRoot returns the root directory of the Git repository
func (r *GitRepository) GetRoot() (string, error) {
	return r.root, nil
}

// GetProjectName returns the project name (basename of root directory)
func (r *GitRepository) GetProjectName() string {
	return r.projectName
}

// IsGitRepository checks if the current directory is in a Git repository
func (r *GitRepository) IsGitRepository() bool {
	_, err := getGitRoot()
	return err == nil
}

// BranchExists checks if a local branch exists
func (r *GitRepository) BranchExists(branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = r.root
	err := cmd.Run()

	if err != nil {
		// Check if it's an exit error (branch doesn't exist)
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}

	return true, nil
}

// RemoteBranchExists checks if a remote branch exists
func (r *GitRepository) RemoteBranchExists(branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	cmd.Dir = r.root
	err := cmd.Run()

	if err != nil {
		// Check if it's an exit error (branch doesn't exist)
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check remote branch existence: %w", err)
	}

	return true, nil
}

// GetCurrentBranch returns the current branch name
func (r *GitRepository) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = r.root
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CreateBranch creates a new branch
func (r *GitRepository) CreateBranch(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Dir = r.root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branch, err)
	}

	return nil
}

// RemoveBranch deletes a local branch
func (r *GitRepository) RemoveBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "branch", flag, branch)
	cmd.Dir = r.root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}

	return nil
}

// RemoveRemoteBranch deletes a remote branch
func (r *GitRepository) RemoveRemoteBranch(branch string) error {
	cmd := exec.Command("git", "push", "origin", "--delete", branch)
	cmd.Dir = r.root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete remote branch %s: %w", branch, err)
	}

	return nil
}

// CreateWorktree creates a new Git worktree
func (r *GitRepository) CreateWorktree(path, branch string, newBranch bool) error {
	var cmd *exec.Cmd

	if newBranch {
		cmd = exec.Command("git", "worktree", "add", "-b", branch, path)
	} else {
		cmd = exec.Command("git", "worktree", "add", path, branch)
	}

	cmd.Dir = r.root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %s", output)
	}

	return nil
}

// RemoveWorktree removes a Git worktree
func (r *GitRepository) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s", output)
	}

	return nil
}

// ListWorktrees returns a list of all worktrees
func (r *GitRepository) ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(string(output))
}

// GetWorktreePath returns the path of a worktree for the given branch
func (r *GitRepository) GetWorktreePath(branch string) (string, error) {
	worktrees, err := r.ListWorktrees()
	if err != nil {
		return "", err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path, nil
		}
	}

	return "", fmt.Errorf("worktree for branch %s not found", branch)
}

// UpdateGitignore adds files to .gitignore
func (r *GitRepository) UpdateGitignore(files []string) error {
	if len(files) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(r.root, ".gitignore")

	// Read existing .gitignore
	var existing []byte
	if _, err := os.Stat(gitignorePath); err == nil {
		existing, _ = os.ReadFile(gitignorePath)
	}

	// Prepare new content
	content := string(existing)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Add separator comment
	content += "\n# Auto-copied files (added by hatcher)\n"

	// Add files
	for _, file := range files {
		content += file + "\n"
	}

	// Write back to .gitignore
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// DeleteBranch deletes a local branch
func (r *GitRepository) DeleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	err := cmd.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("failed to delete branch %s: branch may not exist or has unmerged changes", branch)
		}
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}

	return nil
}

// DeleteRemoteBranch deletes a remote branch
func (r *GitRepository) DeleteRemoteBranch(branch string) error {
	cmd := exec.Command("git", "push", "origin", "--delete", branch)
	cmd.Dir = r.root
	err := cmd.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("failed to delete remote branch %s: branch may not exist on remote", branch)
		}
		return fmt.Errorf("failed to delete remote branch %s: %w", branch, err)
	}

	return nil
}

// getGitRoot returns the root directory of the Git repository
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'
func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	lines := strings.Split(output, "\n")

	var current Worktree
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	// Add the last worktree if exists
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}
