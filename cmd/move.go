package cmd

import (
	"fmt"

	"github.com/keisukeshimizu/hatcher/internal/editor"
	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	switchEditor bool
	yes          bool
	newWindow    bool
)

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move <branch-name>",
	Short: "Move to existing worktree and open in editor",
	Long: `Move to an existing worktree and open it in your preferred editor.

If the worktree doesn't exist, you'll be prompted to create it (use --yes to skip confirmation).

Examples:
  hatcher move feature/user-auth    # Open worktree in new editor window
  hatcher move -s main             # Switch current editor to main worktree
  hatcher move -y new-feature      # Create and open if doesn't exist
  hatcher move --editor cursor ui  # Open in specific editor`,
	Aliases: []string{"mv", "switch", "open"},
	Args:    cobra.ExactArgs(1),
	RunE:    runMove,
}

func init() {
	rootCmd.AddCommand(moveCmd)

	// Flags for move command
	moveCmd.Flags().BoolVarP(&switchEditor, "switch", "s", false, "close current editor and switch to new worktree")
	moveCmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatically create worktree if it doesn't exist")
	moveCmd.Flags().BoolVar(&newWindow, "new-window", true, "open in new window (default)")
	moveCmd.Flags().StringVar(&editor, "editor", "", "specify editor to use (cursor, code)")
}

func runMove(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	if verbose {
		fmt.Printf("🔍 Searching for worktree: %s\n", branchName)
	}

	// Initialize Git repository
	repo, err := git.NewRepository()
	if err != nil {
		return fmt.Errorf("❌ Not in a Git repository: %w", err)
	}

	// Initialize editor detector
	detector := editor.NewDetector()

	// Create mover
	mover := worktree.NewMover(repo, detector)

	// Prepare move options
	options := worktree.MoveOptions{
		BranchName:    branchName,
		SwitchMode:    switchEditor,
		AutoCreate:    yes,
		EditorCommand: editor,
	}

	// Execute move operation
	result, err := mover.MoveToWorktree(options)
	if err != nil {
		return fmt.Errorf("❌ Failed to move to worktree: %w", err)
	}

	// Display results
	if result.CreatedNew {
		fmt.Printf("🆕 Created new worktree: %s\n", result.WorktreePath)
	} else {
		fmt.Printf("✅ Found worktree: %s\n", result.WorktreePath)
	}

	if switchEditor {
		fmt.Printf("🔄 Switched to %s\n", result.EditorUsed)
	} else {
		fmt.Printf("🚀 Opened in %s\n", result.EditorUsed)
	}

	fmt.Printf("📂 Branch: %s\n", result.BranchName)

	return nil
}
