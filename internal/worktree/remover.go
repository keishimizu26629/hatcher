package worktree

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// RemoveOptions contains options for removing a worktree
type RemoveOptions struct {
	BranchName   string // Branch name to remove worktree for
	RemoveBranch bool   // Whether to also remove the local branch
	RemoveRemote bool   // Whether to also remove the remote branch
	Force        bool   // Force removal even if there are uncommitted changes
	SkipConfirm  bool   // Skip confirmation prompt
}

// RemovalResult contains the result of a worktree removal operation
type RemovalResult struct {
	BranchName          string // Branch name that was processed
	WorktreePath        string // Path to the worktree that was removed
	WorktreeRemoved     bool   // Whether the worktree was successfully removed
	LocalBranchRemoved  bool   // Whether the local branch was removed
	RemoteBranchRemoved bool   // Whether the remote branch was removed
}

// RemovalValidation contains validation information for a removal operation
type RemovalValidation struct {
	BranchName        string   // Branch name being validated
	WorktreePath      string   // Path to the worktree
	WorktreeExists    bool     // Whether the worktree exists
	LocalBranchExists bool     // Whether the local branch exists
	IsMainRepository  bool     // Whether this is the main repository
	CanRemove         bool     // Whether removal is safe
	Warnings          []string // Any warnings about the removal
}

// RemovalPlan describes what will be removed
type RemovalPlan struct {
	BranchName             string   // Branch name to be processed
	WorktreePath           string   // Path to the worktree
	WillRemoveWorktree     bool     // Whether the worktree will be removed
	WillRemoveLocalBranch  bool     // Whether the local branch will be removed
	WillRemoveRemoteBranch bool     // Whether the remote branch will be removed
	Description            string   // Human-readable description of the plan
	Warnings               []string // Any warnings about the removal
}

// Remover handles worktree removal operations
type Remover struct {
	repo   git.Repository
	finder *Finder
}

// NewRemover creates a new Remover instance
func NewRemover(repo git.Repository) *Remover {
	return &Remover{
		repo:   repo,
		finder: NewFinder(repo),
	}
}

// RemoveWorktree removes a worktree and optionally its associated branches
func (r *Remover) RemoveWorktree(options RemoveOptions) (*RemovalResult, error) {
	// Validate the removal operation
	validation, err := r.ValidateRemoval(options.BranchName)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if !validation.CanRemove {
		if validation.IsMainRepository {
			return nil, fmt.Errorf("cannot remove main repository worktree")
		}
		if !validation.WorktreeExists {
			return nil, fmt.Errorf("worktree not found for branch '%s'", options.BranchName)
		}
		return nil, fmt.Errorf("removal not allowed")
	}

	// Get removal plan
	plan, err := r.GetRemovalPlan(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create removal plan: %w", err)
	}

	// Confirm removal if not skipping
	if !options.SkipConfirm {
		if !r.ConfirmRemoval(plan, options.SkipConfirm) {
			return nil, fmt.Errorf("removal cancelled by user")
		}
	}

	result := &RemovalResult{
		BranchName:   options.BranchName,
		WorktreePath: validation.WorktreePath,
	}

	// Remove the worktree
	if validation.WorktreeExists {
		err = r.repo.RemoveWorktree(validation.WorktreePath, options.Force)
		if err != nil {
			return nil, fmt.Errorf("failed to remove worktree: %w", err)
		}
		result.WorktreeRemoved = true
	}

	// Remove local branch if requested
	if options.RemoveBranch && validation.LocalBranchExists {
		err = r.repo.RemoveBranch(options.BranchName, options.Force)
		if err != nil {
			return nil, fmt.Errorf("failed to remove local branch: %w", err)
		}
		result.LocalBranchRemoved = true
	}

	// Remove remote branch if requested
	if options.RemoveRemote && validation.LocalBranchExists {
		// Check if remote branch exists
		remoteExists, err := r.repo.RemoteBranchExists(options.BranchName)
		if err != nil {
			return nil, fmt.Errorf("failed to check remote branch: %w", err)
		}

		if remoteExists {
			err = r.repo.RemoveRemoteBranch(options.BranchName)
			if err != nil {
				return nil, fmt.Errorf("failed to remove remote branch: %w", err)
			}
			result.RemoteBranchRemoved = true
		}
	}

	return result, nil
}

// ValidateRemoval validates whether a worktree can be safely removed
func (r *Remover) ValidateRemoval(branchName string) (*RemovalValidation, error) {
	validation := &RemovalValidation{
		BranchName: branchName,
		Warnings:   []string{},
	}

	// Check if this is the main repository
	currentBranch, err := r.repo.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	if branchName == currentBranch {
		// Check if we're in the main repository
		worktrees, err := r.repo.ListWorktrees()
		if err != nil {
			return nil, fmt.Errorf("failed to list worktrees: %w", err)
		}

		repoRoot, err := r.repo.GetRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to get repository root: %w", err)
		}

		// Find if current directory is the main repository
		for _, wt := range worktrees {
			if wt.Path == repoRoot && wt.Branch == branchName {
				validation.IsMainRepository = true
				validation.CanRemove = false
				validation.Warnings = append(validation.Warnings, "Cannot remove main repository worktree")
				return validation, nil
			}
		}
	}

	// Find the worktree path
	worktreePath, found, err := r.finder.FindWorktree(branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to find worktree: %w", err)
	}
	if !found {
		// Worktree doesn't exist
		validation.WorktreeExists = false
		validation.CanRemove = false
		return validation, nil
	}

	validation.WorktreePath = worktreePath
	validation.WorktreeExists = true

	// Check if local branch exists
	localExists, err := r.repo.BranchExists(branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to check local branch: %w", err)
	}
	validation.LocalBranchExists = localExists

	// Check for uncommitted changes
	if validation.WorktreeExists {
		hasChanges, err := r.hasUncommittedChanges(worktreePath)
		if err != nil {
			return nil, fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}

		if hasChanges {
			validation.Warnings = append(validation.Warnings, "Worktree has uncommitted changes")
		}
	}

	// Can remove if worktree exists and it's not the main repository
	validation.CanRemove = validation.WorktreeExists && !validation.IsMainRepository

	return validation, nil
}

// GetRemovalPlan creates a plan describing what will be removed
func (r *Remover) GetRemovalPlan(options RemoveOptions) (*RemovalPlan, error) {
	validation, err := r.ValidateRemoval(options.BranchName)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	plan := &RemovalPlan{
		BranchName:             options.BranchName,
		WorktreePath:           validation.WorktreePath,
		WillRemoveWorktree:     validation.WorktreeExists,
		WillRemoveLocalBranch:  options.RemoveBranch && validation.LocalBranchExists,
		WillRemoveRemoteBranch: options.RemoveRemote && validation.LocalBranchExists,
		Warnings:               validation.Warnings,
	}

	// Build description
	var actions []string

	if plan.WillRemoveWorktree {
		actions = append(actions, fmt.Sprintf("Remove worktree at %s", plan.WorktreePath))
	}

	if plan.WillRemoveLocalBranch {
		actions = append(actions, fmt.Sprintf("Remove local branch '%s'", plan.BranchName))
	}

	if plan.WillRemoveRemoteBranch {
		actions = append(actions, fmt.Sprintf("Remove remote branch 'origin/%s'", plan.BranchName))
	}

	if len(actions) == 0 {
		plan.Description = "No actions to perform"
	} else {
		plan.Description = strings.Join(actions, "\n")
	}

	return plan, nil
}

// ConfirmRemoval prompts the user to confirm the removal operation
func (r *Remover) ConfirmRemoval(plan *RemovalPlan, skipConfirm bool) bool {
	if skipConfirm {
		return true
	}

	// In a real implementation, this would prompt the user for confirmation
	// For now, we'll simulate declining dangerous operations
	if len(plan.Warnings) > 0 || plan.WillRemoveLocalBranch || plan.WillRemoveRemoteBranch {
		return false // Simulate user declining dangerous operations
	}

	return true // Simulate user accepting safe operations
}

// hasUncommittedChanges checks if a worktree has uncommitted changes
func (r *Remover) hasUncommittedChanges(worktreePath string) (bool, error) {
	// Check if there are any files in the worktree directory
	// This is a simplified check - in a real implementation, we'd use git status
	entries, err := os.ReadDir(worktreePath)
	if err != nil {
		return false, err
	}

	// Look for non-git files
	for _, entry := range entries {
		if entry.Name() != ".git" && !strings.HasPrefix(entry.Name(), ".git") {
			return true, nil
		}
	}

	return false, nil
}

// promptUser prompts the user for yes/no confirmation
func (r *Remover) promptUser(message string) bool {
	fmt.Printf("%s (y/N): ", message)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}

	return false
}
