package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/keisukeshimizu/hatcher/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Hatcher configuration",
	Long: `Manage Hatcher configuration files and settings.

This command allows you to initialize, view, edit, and validate
configuration files for both project-specific and global settings.

Examples:
  hch config init                    # Initialize default config
  hch config show                    # Show current configuration
  hch config edit                    # Edit configuration interactively
  hch config validate                # Validate configuration files`,
	Aliases: []string{"cfg"},
}

// configInitCmd initializes configuration
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Hatcher configuration",
	Long: `Initialize Hatcher configuration with default settings.

Creates a new configuration file with sensible defaults for auto-copy
and other Hatcher features.

Examples:
  hch config init                    # Initialize project config
  hch config init --global           # Initialize global config
  hch config init --force            # Overwrite existing config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		force, _ := cmd.Flags().GetBool("force")
		format, _ := cmd.Flags().GetString("format")

		manager := config.NewManager()

		// Load existing config to preserve settings
		var projectPath string
		if !global {
			var err error
			projectPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
		}

		// Check if config already exists
		configPaths := manager.GetConfigPaths(projectPath)
		var existingConfig string
		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				existingConfig = path
				break
			}
		}

		if existingConfig != "" && !force {
			return fmt.Errorf("configuration already exists at %s (use --force to overwrite)", existingConfig)
		}

		// Create default config
		defaultConfig, err := manager.LoadConfig("")
		if err != nil {
			return fmt.Errorf("failed to load default config: %w", err)
		}

		// Save config
		if err := manager.SaveConfig(defaultConfig, projectPath, global); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		var configType string
		if global {
			configType = "global"
		} else {
			configType = "project"
		}

		fmt.Printf("✅ Initialized %s configuration\n", configType)

		if global {
			homeDir, _ := os.UserHomeDir()
			fmt.Printf("📁 Config location: %s\n", filepath.Join(homeDir, ".hatcher", "config.yaml"))
		} else {
			fmt.Printf("📁 Config location: %s\n", filepath.Join(projectPath, ".hatcher-auto-copy.json"))
		}

		return nil
	},
}

// configShowCmd shows current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Display the current Hatcher configuration.

Shows the merged configuration from all sources including defaults,
global config, project config, and environment variables.

Examples:
  hch config show                    # Show current config
  hch config show --format json     # Show as JSON
  hch config show --format yaml     # Show as YAML
  hch config show --paths            # Show config file paths`,
	Aliases: []string{"get", "view"},
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		showPaths, _ := cmd.Flags().GetBool("paths")

		manager := config.NewManager()

		// Get current directory for project config
		projectPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Show config paths if requested
		if showPaths {
			paths := manager.GetConfigPaths(projectPath)
			fmt.Println("📁 Configuration file search paths (in priority order):")
			for i, path := range paths {
				exists := "❌"
				if _, err := os.Stat(path); err == nil {
					exists = "✅"
				}
				fmt.Printf("%d. %s %s\n", i+1, exists, path)
			}
			fmt.Println()
		}

		// Load and display config
		cfg, err := manager.LoadConfig(projectPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		switch format {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(cfg)
		case "yaml":
			encoder := yaml.NewEncoder(os.Stdout)
			defer encoder.Close()
			return encoder.Encode(cfg)
		default:
			return displayConfigTable(cfg)
		}
	},
}

// configEditCmd edits configuration interactively
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration interactively",
	Long: `Edit Hatcher configuration using your default editor.

Opens the configuration file in your preferred editor for modification.
Creates the file if it doesn't exist.

Examples:
  hch config edit                    # Edit project config
  hch config edit --global           # Edit global config
  hch config edit --editor vim       # Use specific editor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		editor, _ := cmd.Flags().GetString("editor")

		manager := config.NewManager()

		var configPath string
		var projectPath string

		if global {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			configDir := filepath.Join(homeDir, ".hatcher")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			configPath = filepath.Join(configDir, "config.yaml")
		} else {
			var err error
			projectPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			configPath = filepath.Join(projectPath, ".hatcher-auto-copy.json")
		}

		// Create config file if it doesn't exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			defaultConfig, err := manager.LoadConfig("")
			if err != nil {
				return fmt.Errorf("failed to load default config: %w", err)
			}

			if err := manager.SaveConfig(defaultConfig, projectPath, global); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}

			fmt.Printf("📝 Created new config file: %s\n", configPath)
		}

		// Determine editor to use
		if editor == "" {
			editor = os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano" // Default fallback
			}
		}

		// Open editor
		fmt.Printf("📝 Opening %s with %s...\n", configPath, editor)

		// This would normally execute the editor
		// For now, just show what would happen
		fmt.Printf("🔧 Would execute: %s %s\n", editor, configPath)
		fmt.Println("💡 After editing, run 'hch config validate' to check your changes")

		return nil
	},
}

// configValidateCmd validates configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long: `Validate Hatcher configuration files for syntax and semantic errors.

Checks all configuration files for proper JSON/YAML syntax, valid settings,
and logical consistency.

Examples:
  hch config validate                # Validate current config
  hch config validate --fix          # Attempt to fix issues automatically`,
	Aliases: []string{"check"},
	RunE: func(cmd *cobra.Command, args []string) error {
		fix, _ := cmd.Flags().GetBool("fix")

		manager := config.NewManager()

		// Get current directory for project config
		projectPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Load configuration
		cfg, err := manager.LoadConfig(projectPath)
		if err != nil {
			fmt.Printf("❌ Configuration loading failed: %v\n", err)
			return err
		}

		// Validate configuration
		errors := manager.ValidateConfig(cfg)
		if len(errors) == 0 {
			fmt.Println("✅ Configuration is valid")
			return nil
		}

		fmt.Printf("❌ Found %d validation error(s):\n", len(errors))
		for i, err := range errors {
			fmt.Printf("%d. %s\n", i+1, err)
		}

		if fix {
			fmt.Println("\n🔧 Attempting to fix issues...")
			// This would implement automatic fixes
			fmt.Println("💡 Automatic fixes not yet implemented")
		} else {
			fmt.Println("\n💡 Use --fix to attempt automatic repairs")
		}

		return fmt.Errorf("configuration validation failed")
	},
}

// displayConfigTable displays configuration in a readable table format
func displayConfigTable(cfg *config.Config) error {
	fmt.Println("📋 Current Hatcher Configuration")
	fmt.Println()

	// Auto-copy settings
	fmt.Println("🔄 Auto-copy Settings:")
	fmt.Printf("  Version: %d\n", cfg.AutoCopy.Version)
	fmt.Printf("  Items: %d\n", len(cfg.AutoCopy.Items))
	for i, item := range cfg.AutoCopy.Items {
		var itemType string
		if item.Directory != nil {
			if *item.Directory {
				itemType = "directory"
			} else {
				itemType = "file"
			}
		} else {
			itemType = "auto-detect"
		}

		fmt.Printf("    %d. %s (%s)", i+1, item.Path, itemType)
		if item.Recursive {
			fmt.Print(" [recursive]")
		}
		if item.RootOnly {
			fmt.Print(" [root-only]")
		}
		fmt.Println()
	}
	fmt.Println()

	// Editor settings
	fmt.Println("📝 Editor Settings:")
	fmt.Printf("  Preferred: %s\n", cfg.Editor.Preferred)
	fmt.Printf("  Auto-switch: %t\n", cfg.Editor.AutoSwitch)
	fmt.Printf("  Window reuse: %t\n", cfg.Editor.WindowReuse)
	fmt.Println()

	// Global settings
	fmt.Println("🌐 Global Settings:")
	fmt.Printf("  Verbose: %t\n", cfg.Global.Verbose)
	fmt.Printf("  Output format: %s\n", cfg.Global.OutputFormat)
	fmt.Printf("  Color output: %t\n", cfg.Global.ColorOutput)

	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)

	// Flags for init command
	configInitCmd.Flags().Bool("global", false, "Initialize global configuration")
	configInitCmd.Flags().BoolP("force", "f", false, "Overwrite existing configuration")
	configInitCmd.Flags().String("format", "json", "Configuration format (json, yaml)")

	// Flags for show command
	configShowCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	configShowCmd.Flags().Bool("paths", false, "Show configuration file paths")

	// Flags for edit command
	configEditCmd.Flags().Bool("global", false, "Edit global configuration")
	configEditCmd.Flags().String("editor", "", "Editor to use (overrides $EDITOR)")

	// Flags for validate command
	configValidateCmd.Flags().Bool("fix", false, "Attempt to fix issues automatically")
}
