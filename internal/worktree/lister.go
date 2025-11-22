package worktree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// ListOptions contains options for listing worktrees
type ListOptions struct {
	ShowAll    bool // Show all worktrees, not just Hatcher-managed ones
	ShowPaths  bool // Show full paths in output
	ShowStatus bool // Show status information (clean/dirty)
}

// ListResult contains the result of listing worktrees
type ListResult struct {
	Worktrees []WorktreeInfo `json:"worktrees"`
	Total     int            `json:"total"`
}

// Lister handles worktree listing operations
type Lister struct {
	repo git.Repository
}

// NewLister creates a new Lister instance
func NewLister(repo git.Repository) *Lister {
	return &Lister{
		repo: repo,
	}
}

// ListWorktrees lists all worktrees based on the provided options
func (l *Lister) ListWorktrees(options ListOptions) (*ListResult, error) {
	// Get all worktrees from Git
	gitWorktrees, err := l.repo.ListWorktrees()
	if err != nil {
		return nil, fmt.Errorf("failed to list Git worktrees: %w", err)
	}

	// Get repository root for comparison
	repoRoot, err := l.repo.GetRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	var worktrees []WorktreeInfo

	for _, gitWt := range gitWorktrees {
		wtInfo := WorktreeInfo{
			Branch: gitWt.Branch,
			Path:   gitWt.Path,
			Head:   gitWt.Head,
			IsMain: gitWt.Path == repoRoot,
		}

		// Determine if this is Hatcher-managed
		wtInfo.IsHatcherManaged = l.isHatcherManaged(gitWt.Path, gitWt.Branch)

		// Get status if requested
		if options.ShowStatus {
			status, err := l.GetWorktreeStatus(gitWt.Path)
			if err != nil {
				// Don't fail the entire operation for status errors
				status = git.StatusUnknown
			}
			wtInfo.Status = status
		}

		// Filter based on options
		if !options.ShowAll && !wtInfo.IsHatcherManaged && !wtInfo.IsMain {
			continue // Skip non-Hatcher worktrees when ShowAll is false
		}

		worktrees = append(worktrees, wtInfo)
	}

	// Sort worktrees by branch name
	sort.Slice(worktrees, func(i, j int) bool {
		// Main repository first
		if worktrees[i].IsMain && !worktrees[j].IsMain {
			return true
		}
		if !worktrees[i].IsMain && worktrees[j].IsMain {
			return false
		}
		// Then by branch name
		return worktrees[i].Branch < worktrees[j].Branch
	})

	return &ListResult{
		Worktrees: worktrees,
		Total:     len(worktrees),
	}, nil
}

// GetWorktreeStatus gets the status of a specific worktree
func (l *Lister) GetWorktreeStatus(worktreePath string) (git.WorktreeStatus, error) {
	// This is a simplified implementation
	// In a real implementation, we'd check git status in the worktree directory

	// For now, assume clean status for existing directories
	return git.StatusClean, nil
}

// isHatcherManaged determines if a worktree is managed by Hatcher
func (l *Lister) isHatcherManaged(worktreePath, branchName string) bool {
	// Get project name
	projectName := l.repo.GetProjectName()

	// Check if the path follows Hatcher naming convention
	expectedName := fmt.Sprintf("%s-%s", projectName, SanitizeBranchName(branchName))
	actualName := filepath.Base(worktreePath)

	return actualName == expectedName
}

// FormatAsTable formats the result as a table
func (r *ListResult) FormatAsTable() string {
	if len(r.Worktrees) == 0 {
		return "No worktrees found.\n"
	}

	var output bytes.Buffer
	w := tabwriter.NewWriter(&output, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(w, "BRANCH\tPATH\tSTATUS\tTYPE")
	fmt.Fprintln(w, "------\t----\t------\t----")

	// Rows
	for _, wt := range r.Worktrees {
		var wtType string
		if wt.IsMain {
			wtType = "main"
		} else if wt.IsHatcherManaged {
			wtType = "hatcher"
		} else {
			wtType = "manual"
		}

		status := string(wt.Status)
		if status == "" {
			status = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", wt.Branch, wt.Path, status, wtType)
	}

	w.Flush()
	return output.String()
}

// FormatAsJSON formats the result as JSON
func (r *ListResult) FormatAsJSON() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal JSON: %s"}`, err.Error())
	}
	return string(data)
}

// FormatAsSimple formats the result as a simple list
func (r *ListResult) FormatAsSimple() string {
	if len(r.Worktrees) == 0 {
		return "No worktrees found.\n"
	}

	var output strings.Builder

	for _, wt := range r.Worktrees {
		var prefix string
		if wt.IsMain {
			prefix = "* "
		} else {
			prefix = "  "
		}

		fmt.Fprintf(&output, "%s%s\n", prefix, wt.Branch)
	}

	return output.String()
}

// FilterByBranchPattern filters worktrees by branch name pattern
func (r *ListResult) FilterByBranchPattern(pattern string) []WorktreeInfo {
	var filtered []WorktreeInfo

	// Convert simple glob pattern to basic matching
	// This is a simplified implementation - a real one would use proper glob matching
	prefix := strings.TrimSuffix(pattern, "*")

	for _, wt := range r.Worktrees {
		if strings.HasPrefix(wt.Branch, prefix) {
			filtered = append(filtered, wt)
		}
	}

	return filtered
}

// FilterByStatus filters worktrees by status
func (r *ListResult) FilterByStatus(status git.WorktreeStatus) []WorktreeInfo {
	var filtered []WorktreeInfo

	for _, wt := range r.Worktrees {
		if wt.Status == status {
			filtered = append(filtered, wt)
		}
	}

	return filtered
}

// FilterHatcherManaged filters to show only Hatcher-managed worktrees
func (r *ListResult) FilterHatcherManaged() []WorktreeInfo {
	var filtered []WorktreeInfo

	for _, wt := range r.Worktrees {
		if wt.IsHatcherManaged || wt.IsMain {
			filtered = append(filtered, wt)
		}
	}

	return filtered
}
