package cmd

import (
	"fmt"
	"os"

	"github.com/keisukeshimizu/hatcher/internal/doctor"
	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and validate Hatcher configuration",
	Long: `Diagnose system configuration and validate Hatcher setup.

Checks Git configuration, editor availability, configuration files, and system requirements.

Examples:
  hch doctor                    # Run all diagnostic checks
  hch doctor --format json     # Output results in JSON format
  hch doctor --simple          # Use simple output format`,
	Aliases: []string{"check", "validate", "diagnose"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		outputFormat, _ := cmd.Flags().GetString("format")
		useSimple, _ := cmd.Flags().GetBool("simple")

		// Initialize Git repository (optional for doctor)
		var repo git.Repository
		var err error

		// Try to initialize repository, but don't fail if not in a Git repo
		repo, err = git.NewRepositoryFromPath(".")
		if err != nil {
			// Not in a Git repository - that's okay for doctor command
			repo = nil
		}

		// Create checker
		checker := doctor.NewChecker(repo)

		// Run diagnostic checks
		result, err := checker.CheckSystem()
		if err != nil {
			return fmt.Errorf("diagnostic checks failed: %w", err)
		}

		// Output results in requested format
		switch outputFormat {
		case "json":
			fmt.Print(result.FormatAsJSON())
		case "simple":
			fmt.Print(result.FormatAsSimple())
		default:
			if useSimple {
				fmt.Print(result.FormatAsSimple())
			} else {
				fmt.Print(result.FormatAsTable())
			}
		}

		// Exit with appropriate code based on overall status
		overallStatus := result.GetOverallStatus()
		switch overallStatus {
		case doctor.CheckStatusFail:
			os.Exit(1)
		case doctor.CheckStatusWarn:
			os.Exit(2)
		default:
			os.Exit(0)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	// Add flags
	doctorCmd.Flags().StringP("format", "f", "table", "Output format (table, json, simple)")
	doctorCmd.Flags().Bool("simple", false, "Use simple output format")
}
