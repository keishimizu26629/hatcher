package cmd

import (
	"fmt"
	"os"

	"github.com/keisukeshimizu/hatcher/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	verbose   bool
	dryRun    bool
	noColor   bool
	configDir string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatcher",
	Short: "🥇 Git worktree management tool that hatches AI-powered development environments",
	Long: `🥇 Hatcher - Git Worktree Tool

A powerful command-line tool that simplifies Git worktree management with automatic
directory naming, branch detection, and editor integration.

Hatcher "hatches" your worktrees into AI-powered development environments by:
- Creating worktrees with consistent naming (project-branch-name)
- Auto-copying AI configuration files (.ai/, .cursorrules, etc.)
- Integrating with editors (Cursor, VS Code)
- Managing the complete worktree lifecycle

Examples:
  hatcher feature/user-auth     # Create worktree for feature branch
  hatcher move main            # Switch to main worktree in editor
  hatcher remove old-feature   # Remove completed worktree
  hatcher list                 # Show all managed worktrees`,
	Version: "1.0.0",
	// Default command: create worktree
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Update logger verbose setting
		logger.UpdateVerbose()
		
		if len(args) == 1 {
			// If branch name is provided, run create command
			return runCreate(cmd, args)
		}
		// Otherwise show help
		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/hatcher/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", false, "show what would be done without executing")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "config directory path")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("config-dir", rootCmd.PersistentFlags().Lookup("config-dir"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".hatcher" (without extension).
		viper.AddConfigPath(home + "/.config/hatcher")
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
