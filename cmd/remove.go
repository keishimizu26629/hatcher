package cmd

import (
	"fmt"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/internal/worktree"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [branch-name]",
	Short: "Remove a worktree and optionally its branch",
	Long: `Remove a Git worktree and optionally its associated local and remote branches.

This command provides safe removal of worktrees with confirmation prompts
and validation to prevent accidental data loss.

Examples:
  hch remove feature/new-ui              # Remove worktree only
  hch remove feature/new-ui --branch     # Remove worktree and local branch
  hch remove feature/new-ui --all        # Remove worktree, local and remote branch
  hch remove feature/new-ui --force      # Force removal even with uncommitted changes
  hch remove feature/new-ui --yes        # Skip confirmation prompt`,
	Aliases: []string{"rm", "delete", "del"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		// Get flags
		removeBranch, _ := cmd.Flags().GetBool("branch")
		removeAll, _ := cmd.Flags().GetBool("all")
		force, _ := cmd.Flags().GetBool("force")
		skipConfirm, _ := cmd.Flags().GetBool("yes")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// If --all is specified, remove both local and remote branches
		removeRemote := removeAll
		if removeAll {
			removeBranch = true
		}

		// Initialize Git repository
		repo, err := git.NewRepositoryFromPath(".")
		if err != nil {
			return fmt.Errorf("failed to initialize Git repository: %w", err)
		}

		// Create remover
		remover := worktree.NewRemover(repo)

		// Prepare options
		options := worktree.RemoveOptions{
			BranchName:   branchName,
			RemoveBranch: removeBranch,
			RemoveRemote: removeRemote,
			Force:        force,
			SkipConfirm:  skipConfirm,
		}

		// Dry run mode
		if dryRun {
			plan, err := remover.GetRemovalPlan(options)
			if err != nil {
				return fmt.Errorf("failed to create removal plan: %w", err)
			}

			fmt.Printf("Dry run mode - would perform the following actions:\n\n")
			fmt.Printf("Branch: %s\n", plan.BranchName)
			if plan.WorktreePath != "" {
				fmt.Printf("Worktree: %s\n", plan.WorktreePath)
			}
			fmt.Printf("\nActions:\n%s\n", plan.Description)

			if len(plan.Warnings) > 0 {
				fmt.Printf("\nWarnings:\n")
				for _, warning := range plan.Warnings {
					fmt.Printf("  ⚠️  %s\n", warning)
				}
			}

			return nil
		}

		// Perform removal
		result, err := remover.RemoveWorktree(options)
		if err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}

		// Output result
		fmt.Printf("✅ Successfully processed branch '%s'\n\n", result.BranchName)

		if result.WorktreeRemoved {
			fmt.Printf("🗂️  Removed worktree: %s\n", result.WorktreePath)
		}

		if result.LocalBranchRemoved {
			fmt.Printf("🌿 Removed local branch: %s\n", result.BranchName)
		}

		if result.RemoteBranchRemoved {
			fmt.Printf("☁️  Removed remote branch: origin/%s\n", result.BranchName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Add flags
	removeCmd.Flags().BoolP("branch", "b", false, "Also remove the local branch")
	removeCmd.Flags().Bool("all", false, "Remove worktree, local branch, and remote branch")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal even if there are uncommitted changes")
	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	removeCmd.Flags().Bool("dry-run", false, "Show what would be removed without actually removing")
}
