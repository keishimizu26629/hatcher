package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/keisukeshimizu/hatcher/internal/autocopy"
	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/internal/logger"
	"github.com/keisukeshimizu/hatcher/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	noCopy            bool
	noGitignoreUpdate bool
	force             bool
	editor            string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a new worktree for the specified branch",
	Long: `Create a new Git worktree with automatic directory naming and file copying.

The worktree will be created in the parent directory of the current Git repository
with the naming pattern: {project-name}-{branch-name-safe}

Examples:
  hatcher create feature/user-auth    # Creates: ../myapp-feature-user-auth
  hatcher feature/user-auth           # Same as above (default command)
  hatcher create --no-copy main       # Skip auto file copying
  hatcher create --force test         # Overwrite existing directory`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Flags for create command
	createCmd.Flags().BoolVar(&noCopy, "no-copy", false, "skip automatic file copying")
	createCmd.Flags().BoolVar(&noGitignoreUpdate, "no-gitignore-update", false, "skip .gitignore update")
	createCmd.Flags().BoolVar(&force, "force", false, "force overwrite existing directory")
	createCmd.Flags().StringVar(&editor, "editor", "", "open in specified editor after creation (cursor, code)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	branchName := args[0]
	
	// Update logger verbose setting
	logger.UpdateVerbose()
	log := logger.GetLogger()

	log.Debug("Starting worktree creation process")
	log.Verbose("Branch name: %s", branchName)
	log.Verbose("Flags - Force: %t, NoCopy: %t, NoGitignoreUpdate: %t, DryRun: %t", force, noCopy, noGitignoreUpdate, dryRun)

	if verbose {
		fmt.Printf("🔍 Creating worktree for branch '%s'\n", branchName)
	}

	// Initialize Git repository
	log.Debug("Initializing Git repository")
	repo, err := git.NewRepository()
	if err != nil {
		log.Error("Failed to initialize Git repository: %v", err)
		return fmt.Errorf("❌ Not in a Git repository: %w", err)
	}
	log.Debug("Git repository initialized successfully")

	// Create worktree creator
	creator := worktree.NewCreator(repo)

	// Prepare creation options
	opts := worktree.CreateOptions{
		BranchName:        branchName,
		Force:             force,
		NoCopy:            noCopy,
		NoGitignoreUpdate: noGitignoreUpdate,
		DryRun:            dryRun,
	}

	fmt.Printf("📁 Target directory: %s\n", worktree.GenerateWorktreePath(
		func() string { root, _ := repo.GetRoot(); return root }(),
		repo.GetProjectName(),
		branchName,
	))

	// Create the worktree
	result, err := creator.Create(opts)
	if err != nil {
		return fmt.Errorf("❌ Failed to create worktree: %w", err)
	}

	if dryRun {
		fmt.Println("🔍 Dry run mode - showing what would be done:")
		fmt.Printf("  - %s\n", result.Message)
		if result.IsNewBranch {
			fmt.Printf("  - Create new branch: %s\n", result.BranchName)
		} else {
			fmt.Printf("  - Use existing branch: %s\n", result.BranchName)
		}
		if !noCopy {
			fmt.Println("  - Copy configuration files")
		}
		if !noGitignoreUpdate {
			fmt.Println("  - Update .gitignore")
		}
		return nil
	}

	// Show creation result
	if result.IsNewBranch {
		fmt.Printf("🆕 Created new branch: %s\n", result.BranchName)
	} else {
		fmt.Printf("🔍 Using existing branch: %s\n", result.BranchName)
	}
	fmt.Printf("✅ %s\n", result.Message)

	// Auto-copy files if enabled
	if !noCopy {
		root, _ := repo.GetRoot()
		if err := autoCopyFiles(root, result.WorktreePath); err != nil {
			fmt.Printf("⚠️  Auto-copy failed: %v\n", err)
		}
	}

	// Open in editor if specified
	if editor != "" {
		if err := openInEditor(result.WorktreePath, editor); err != nil {
			fmt.Printf("⚠️  Failed to open in editor: %v\n", err)
		}
	}

	// Change to the new directory (print for shell evaluation)
	fmt.Printf("📂 cd %s\n", result.WorktreePath)

	return nil
}

// autoCopyFiles copies configuration files to the new worktree
func autoCopyFiles(srcRoot, worktreePath string) error {
	if verbose {
		fmt.Println("📋 Auto-copying configuration files...")
	}

	// Define configuration file paths in priority order
	configPaths := []string{
		filepath.Join(srcRoot, ".vscode", "auto-copy-files.json"),
		filepath.Join(srcRoot, ".worktree-files", "auto-copy-files.json"),
		filepath.Join(os.Getenv("HOME"), ".config", "git", "worktree-files", "auto-copy-files.json"),
	}

	// Load configuration
	config, err := autocopy.LoadAutoCopyConfig(configPaths)
	if err != nil {
		return fmt.Errorf("failed to load auto-copy configuration: %w", err)
	}

	// Validate configuration
	if err := autocopy.ValidateAutoCopyConfig(config); err != nil {
		return fmt.Errorf("invalid auto-copy configuration: %w", err)
	}

	// Skip if no configuration found
	if config.Version == 0 && len(config.Items) == 0 && len(config.Files) == 0 {
		if verbose {
			fmt.Println("ℹ️  No auto-copy configuration found, skipping file copying")
		}
		return nil
	}

	// Create auto-copier and copy files
	copier := autocopy.NewLegacyAutoCopier()
	copiedFiles, err := copier.CopyFiles(srcRoot, worktreePath, config)
	if err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	if len(copiedFiles) > 0 {
		fmt.Printf("📋 Auto-copied %d files/directories:\n", len(copiedFiles))
		for _, file := range copiedFiles {
			fmt.Printf("  ✅ %s\n", file)
		}

		// Update .gitignore if not disabled
		if !noGitignoreUpdate {
			if err := copier.UpdateGitignore(worktreePath, copiedFiles); err != nil {
				fmt.Printf("⚠️  Failed to update .gitignore: %v\n", err)
			} else {
				fmt.Printf("  ✅ Updated .gitignore with %d entries\n", len(copiedFiles))
			}
		}
	} else {
		if verbose {
			fmt.Println("ℹ️  No files matched auto-copy configuration")
		}
	}

	return nil
}

// openInEditor opens the worktree in the specified editor
func openInEditor(path, editorName string) error {
	fmt.Printf("🚀 Opening in %s...\n", editorName)
	// Placeholder implementation
	// This will be replaced with actual editor integration
	return nil
}
