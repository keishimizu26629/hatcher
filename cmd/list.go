package cmd

import (
	"fmt"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/internal/worktree"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed worktrees",
	Long: `List all worktrees managed by Hatcher with their status and information.

By default, only shows Hatcher-managed worktrees. Use --all to show all Git worktrees.

Examples:
  hch list                          # Show Hatcher-managed worktrees
  hch list --all                    # Show all Git worktrees
  hch list --format json           # Output in JSON format
  hch list --filter "feature/*"    # Filter by branch pattern
  hch list --paths                  # Show full paths`,
	Aliases: []string{"ls", "show"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		showAll, _ := cmd.Flags().GetBool("all")
		showPaths, _ := cmd.Flags().GetBool("paths")
		showStatus, _ := cmd.Flags().GetBool("status")
		outputFormat, _ := cmd.Flags().GetString("format")
		filterPattern, _ := cmd.Flags().GetString("filter")

		// Initialize Git repository
		repo, err := git.NewRepositoryFromPath(".")
		if err != nil {
			return fmt.Errorf("failed to initialize Git repository: %w", err)
		}

		// Create lister
		lister := worktree.NewLister(repo)

		// Prepare options
		options := worktree.ListOptions{
			ShowAll:    showAll,
			ShowPaths:  showPaths,
			ShowStatus: showStatus,
		}

		// List worktrees
		result, err := lister.ListWorktrees(options)
		if err != nil {
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		// Apply filter if specified
		if filterPattern != "" {
			filtered := result.FilterByBranchPattern(filterPattern)
			result.Worktrees = filtered
			result.Total = len(filtered)
		}

		// Output in requested format
		switch outputFormat {
		case "json":
			fmt.Print(result.FormatAsJSON())
		case "simple":
			fmt.Print(result.FormatAsSimple())
		case "table":
			fallthrough
		default:
			fmt.Print(result.FormatAsTable())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().Bool("all", false, "Show all Git worktrees, not just Hatcher-managed ones")
	listCmd.Flags().Bool("paths", false, "Show full paths in output")
	listCmd.Flags().Bool("status", false, "Show status information (clean/dirty)")
	listCmd.Flags().StringP("format", "f", "table", "Output format (table, json, simple)")
	listCmd.Flags().String("filter", "", "Filter worktrees by branch pattern (e.g., 'feature/*')")
}
