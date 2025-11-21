package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// Creator handles worktree creation logic
type Creator struct {
	repo git.Repository
}

// NewCreator creates a new worktree creator
func NewCreator(repo git.Repository) *Creator {
	return &Creator{
		repo: repo,
	}
}

// CreateOptions contains options for worktree creation
type CreateOptions struct {
	BranchName        string
	Force             bool
	NoCopy            bool
	NoGitignoreUpdate bool
	DryRun            bool
}

// CreateResult contains the result of worktree creation
type CreateResult struct {
	WorktreePath string
	BranchName   string
	IsNewBranch  bool
	Message      string
}

// Create creates a new worktree with the specified options
func (c *Creator) Create(opts CreateOptions) (*CreateResult, error) {
	// Validate branch name
	if err := ValidateBranchName(opts.BranchName); err != nil {
		return nil, fmt.Errorf("invalid branch name: %w", err)
	}

	// Get repository information
	root, err := c.repo.GetRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	projectName := c.repo.GetProjectName()
	branchNameSafe := SanitizeBranchName(opts.BranchName)
	dirName := fmt.Sprintf("%s-%s", projectName, branchNameSafe)

	parentDir := filepath.Dir(root)
	worktreePath := filepath.Join(parentDir, dirName)

	// Check if directory already exists
	if _, err := os.Stat(worktreePath); err == nil && !opts.Force {
		return nil, fmt.Errorf("directory already exists: %s (use --force to overwrite)", worktreePath)
	}

	// Determine if we need to create a new branch
	localExists, err := c.repo.BranchExists(opts.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to check local branch existence: %w", err)
	}

	remoteExists, err := c.repo.RemoteBranchExists(opts.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to check remote branch existence: %w", err)
	}

	isNewBranch := !localExists && !remoteExists

	if opts.DryRun {
		return &CreateResult{
			WorktreePath: worktreePath,
			BranchName:   opts.BranchName,
			IsNewBranch:  isNewBranch,
			Message:      fmt.Sprintf("Would create worktree at: %s", worktreePath),
		}, nil
	}

	// Remove existing directory if force is enabled
	if opts.Force {
		if err := os.RemoveAll(worktreePath); err != nil {
			return nil, fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Create the worktree
	if err := c.repo.CreateWorktree(worktreePath, opts.BranchName, isNewBranch); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	result := &CreateResult{
		WorktreePath: worktreePath,
		BranchName:   opts.BranchName,
		IsNewBranch:  isNewBranch,
		Message:      fmt.Sprintf("Worktree created: %s", worktreePath),
	}

	return result, nil
}

// ValidateBranchName validates a branch name for security and compatibility
func ValidateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check for dangerous characters
	dangerous := []string{"..", "|", "&", ";", "$", "`", "\\"}
	for _, char := range dangerous {
		if strings.Contains(branch, char) {
			return fmt.Errorf("branch name contains dangerous character: %s", char)
		}
	}

	// Length check
	if len(branch) > 100 {
		return fmt.Errorf("branch name too long (max 100 characters)")
	}

	// Check for invalid Git branch name patterns
	if strings.HasPrefix(branch, "-") || strings.HasSuffix(branch, ".") {
		return fmt.Errorf("invalid branch name format")
	}

	// Check for consecutive dots or slashes
	if strings.Contains(branch, "..") || strings.Contains(branch, "//") {
		return fmt.Errorf("branch name contains invalid consecutive characters")
	}

	return nil
}

// SanitizeBranchName converts a branch name to a filesystem-safe format
func SanitizeBranchName(branch string) string {
	// Replace / with -
	safe := strings.ReplaceAll(branch, "/", "-")

	// Replace other problematic characters
	replacements := map[string]string{
		" ":  "-",
		"@":  "-",
		"#":  "-",
		":":  "-",
		"*":  "-",
		"?":  "-",
		"\"": "-",
		"<":  "-",
		">":  "-",
		"|":  "-",
	}

	for old, new := range replacements {
		safe = strings.ReplaceAll(safe, old, new)
	}

	// Remove leading/trailing dashes
	safe = strings.Trim(safe, "-")

	// Collapse multiple consecutive dashes
	for strings.Contains(safe, "--") {
		safe = strings.ReplaceAll(safe, "--", "-")
	}

	return safe
}

// GenerateWorktreePath generates the full path for a worktree
func GenerateWorktreePath(repoRoot, projectName, branchName string) string {
	branchNameSafe := SanitizeBranchName(branchName)
	dirName := fmt.Sprintf("%s-%s", projectName, branchNameSafe)
	parentDir := filepath.Dir(repoRoot)
	return filepath.Join(parentDir, dirName)
}

// IsHatcherWorktree checks if a worktree was created by Hatcher based on naming convention
func IsHatcherWorktree(worktreePath, projectName string) bool {
	dirName := filepath.Base(worktreePath)
	expectedPrefix := projectName + "-"
	return strings.HasPrefix(dirName, expectedPrefix)
}
