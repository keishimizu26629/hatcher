package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// Finder handles worktree discovery and management
type Finder struct {
	repo git.Repository
}

// NewFinder creates a new worktree finder
func NewFinder(repo git.Repository) *Finder {
	return &Finder{
		repo: repo,
	}
}

// WorktreeInfo represents information about a worktree
type WorktreeInfo struct {
	Branch    string    `json:"branch"`
	Path      string    `json:"path"`
	Head      string    `json:"head"`
	Status    string    `json:"status"`
	Created   time.Time `json:"created"`
	IsHatcher bool      `json:"isHatcher"`
	Editor    string    `json:"editor,omitempty"`
}

// WorktreeStatus represents the status of a worktree
type WorktreeStatus string

const (
	StatusClean   WorktreeStatus = "clean"
	StatusDirty   WorktreeStatus = "dirty"
	StatusActive  WorktreeStatus = "active"
	StatusUnknown WorktreeStatus = "unknown"
)

// FindWorktree finds a worktree for the given branch name
func (f *Finder) FindWorktree(branchName string) (string, bool, error) {
	// Get all worktrees
	worktrees, err := f.repo.ListWorktrees()
	if err != nil {
		return "", false, fmt.Errorf("failed to list worktrees: %w", err)
	}

	projectName := f.repo.GetProjectName()
	expectedPath := GenerateWorktreePath(
		func() string { root, _ := f.repo.GetRoot(); return root }(),
		projectName,
		branchName,
	)

	// First, try to find by exact branch match
	for _, wt := range worktrees {
		if wt.Branch == branchName {
			return wt.Path, true, nil
		}
	}

	// Second, try to find by expected path (for hatcher-created worktrees)
	for _, wt := range worktrees {
		if wt.Path == expectedPath {
			return wt.Path, true, nil
		}
	}

	// Third, try to find by hatcher naming convention
	for _, wt := range worktrees {
		if IsHatcherWorktree(wt.Path, projectName) {
			// Extract branch name from path and compare
			if f.extractBranchFromPath(wt.Path, projectName) == branchName {
				return wt.Path, true, nil
			}
		}
	}

	return "", false, nil
}

// ListHatcherWorktrees returns all worktrees managed by hatcher
func (f *Finder) ListHatcherWorktrees() ([]WorktreeInfo, error) {
	// Get all worktrees from Git
	gitWorktrees, err := f.repo.ListWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var hatcherWorktrees []WorktreeInfo
	projectName := f.repo.GetProjectName()

	for _, gitWt := range gitWorktrees {
		info, err := f.convertToWorktreeInfo(gitWt, projectName)
		if err != nil {
			// Log error but continue with other worktrees
			continue
		}
		hatcherWorktrees = append(hatcherWorktrees, *info)
	}

	return hatcherWorktrees, nil
}

// GetWorktreeInfo returns detailed information about a specific worktree
func (f *Finder) GetWorktreeInfo(worktreePath string) (*WorktreeInfo, error) {
	// Check if path exists
	if _, err := os.Stat(worktreePath); err != nil {
		return nil, fmt.Errorf("worktree path does not exist: %s", worktreePath)
	}

	// Get all worktrees and find the matching one
	gitWorktrees, err := f.repo.ListWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	projectName := f.repo.GetProjectName()

	for _, gitWt := range gitWorktrees {
		if gitWt.Path == worktreePath {
			return f.convertToWorktreeInfo(gitWt, projectName)
		}
	}

	return nil, fmt.Errorf("worktree not found in Git worktree list: %s", worktreePath)
}

// convertToWorktreeInfo converts a Git worktree to WorktreeInfo
func (f *Finder) convertToWorktreeInfo(gitWt git.Worktree, projectName string) (*WorktreeInfo, error) {
	// Determine if this is a hatcher-managed worktree
	isHatcher := IsHatcherWorktree(gitWt.Path, projectName)

	// Get file modification time as creation time approximation
	var created time.Time
	if stat, err := os.Stat(gitWt.Path); err == nil {
		created = stat.ModTime()
	} else {
		created = time.Now()
	}

	// Determine status (simplified for now)
	status := string(StatusClean)
	if gitWt.Status == git.StatusDirty {
		status = string(StatusDirty)
	} else if gitWt.Status == git.StatusActive {
		status = string(StatusActive)
	}

	return &WorktreeInfo{
		Branch:    gitWt.Branch,
		Path:      gitWt.Path,
		Head:      gitWt.Head,
		Status:    status,
		Created:   created,
		IsHatcher: isHatcher,
		Editor:    "", // Will be populated by editor detection
	}, nil
}

// extractBranchFromPath extracts the branch name from a hatcher worktree path
func (f *Finder) extractBranchFromPath(worktreePath, projectName string) string {
	dirName := filepath.Base(worktreePath)
	expectedPrefix := projectName + "-"

	if !strings.HasPrefix(dirName, expectedPrefix) {
		return ""
	}

	// Remove project prefix and convert back to branch name
	branchPart := strings.TrimPrefix(dirName, expectedPrefix)

	// Convert sanitized name back to original branch name (best effort)
	// This is a simplified reverse conversion
	branchName := strings.ReplaceAll(branchPart, "-", "/")

	// Handle common patterns
	if strings.HasPrefix(branchName, "feature/") ||
		strings.HasPrefix(branchName, "bugfix/") ||
		strings.HasPrefix(branchName, "hotfix/") ||
		strings.HasPrefix(branchName, "release/") {
		return branchName
	}

	// If no clear pattern, return as-is with dashes converted to slashes
	return branchName
}

// IsHatcherWorktree checks if a worktree was created by Hatcher
func IsHatcherWorktree(worktreePath, projectName string) bool {
	dirName := filepath.Base(worktreePath)
	expectedPrefix := projectName + "-"

	// Must start with project name followed by dash
	if !strings.HasPrefix(dirName, expectedPrefix) {
		return false
	}

	// Must not be the main repository (exact project name)
	if dirName == projectName {
		return false
	}

	// Must have something after the project name and dash
	suffix := strings.TrimPrefix(dirName, expectedPrefix)
	return len(suffix) > 0
}

// GenerateWorktreePath generates the expected path for a hatcher worktree
func GenerateWorktreePath(repoRoot, projectName, branchName string) string {
	branchNameSafe := SanitizeBranchName(branchName)
	dirName := fmt.Sprintf("%s-%s", projectName, branchNameSafe)
	parentDir := filepath.Dir(repoRoot)
	return filepath.Join(parentDir, dirName)
}
