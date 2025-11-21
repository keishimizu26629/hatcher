package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// CheckStatus represents the status of a diagnostic check
type CheckStatus string

const (
	CheckStatusPass CheckStatus = "pass"
	CheckStatusWarn CheckStatus = "warn"
	CheckStatusFail CheckStatus = "fail"
)

// CheckResult represents the result of a single diagnostic check
type CheckResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      CheckStatus `json:"status"`
	Details     string      `json:"details"`
	Suggestions []string    `json:"suggestions,omitempty"`
}

// DiagnosticSummary provides an overview of all checks
type DiagnosticSummary struct {
	Total   int  `json:"total"`
	Passed  int  `json:"passed"`
	Warned  int  `json:"warned"`
	Failed  int  `json:"failed"`
	Healthy bool `json:"healthy"`
}

// DiagnosticResult contains the results of all diagnostic checks
type DiagnosticResult struct {
	Checks  []CheckResult     `json:"checks"`
	Summary DiagnosticSummary `json:"summary"`
}

// Checker performs system diagnostic checks
type Checker struct {
	repo git.Repository
}

// NewChecker creates a new Checker instance
func NewChecker(repo git.Repository) *Checker {
	return &Checker{
		repo: repo,
	}
}

// CheckSystem runs all diagnostic checks
func (c *Checker) CheckSystem() (*DiagnosticResult, error) {
	var checks []CheckResult

	// Run all checks
	checks = append(checks, c.CheckGitInstallation())

	if c.repo != nil {
		checks = append(checks, c.CheckGitRepository())
		checks = append(checks, c.CheckWorktrees())
		checks = append(checks, c.CheckConfiguration())
		checks = append(checks, c.CheckPermissions())
	}

	checks = append(checks, c.CheckEditors())

	// Calculate summary
	summary := c.calculateSummary(checks)

	return &DiagnosticResult{
		Checks:  checks,
		Summary: summary,
	}, nil
}

// CheckGitInstallation checks if Git is properly installed
func (c *Checker) CheckGitInstallation() CheckResult {
	result := CheckResult{
		Name:        "Git Installation",
		Description: "Verify Git is installed and accessible",
	}

	// Check if git command is available
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		result.Status = CheckStatusFail
		result.Details = "Git is not installed or not in PATH"
		result.Suggestions = []string{
			"Install Git from https://git-scm.com/",
			"Ensure Git is in your system PATH",
		}
		return result
	}

	// Parse version
	version := strings.TrimSpace(string(output))
	result.Status = CheckStatusPass
	result.Details = fmt.Sprintf("Git is installed: %s", version)

	return result
}

// CheckGitRepository checks the current Git repository
func (c *Checker) CheckGitRepository() CheckResult {
	result := CheckResult{
		Name:        "Git Repository",
		Description: "Verify current directory is a valid Git repository",
	}

	if c.repo == nil {
		result.Status = CheckStatusFail
		result.Details = "No Git repository context available"
		return result
	}

	// Check if it's a Git repository
	if !c.repo.IsGitRepository() {
		result.Status = CheckStatusFail
		result.Details = "Current directory is not a Git repository"
		result.Suggestions = []string{
			"Navigate to a Git repository",
			"Initialize a new Git repository with 'git init'",
		}
		return result
	}

	// Get repository information
	root, err := c.repo.GetRoot()
	if err != nil {
		result.Status = CheckStatusWarn
		result.Details = "Could not determine repository root"
		return result
	}

	// Check for worktrees
	worktrees, err := c.repo.ListWorktrees()
	if err != nil {
		result.Status = CheckStatusWarn
		result.Details = fmt.Sprintf("Repository found at %s, but could not list worktrees", root)
		return result
	}

	result.Status = CheckStatusPass
	result.Details = fmt.Sprintf("Valid Git repository at %s with %d worktrees", root, len(worktrees))

	return result
}

// CheckWorktrees checks the status of Git worktrees
func (c *Checker) CheckWorktrees() CheckResult {
	result := CheckResult{
		Name:        "Worktrees",
		Description: "Check Git worktrees for issues",
	}

	if c.repo == nil {
		result.Status = CheckStatusFail
		result.Details = "No Git repository available"
		return result
	}

	// List worktrees
	worktrees, err := c.repo.ListWorktrees()
	if err != nil {
		result.Status = CheckStatusFail
		result.Details = "Failed to list worktrees"
		result.Suggestions = []string{
			"Check Git repository integrity",
			"Run 'git worktree prune' to clean up stale entries",
		}
		return result
	}

	// Check each worktree
	var issues []string
	var warnings []string

	for _, wt := range worktrees {
		// Check if directory exists
		if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Worktree directory missing: %s", wt.Path))
		}
	}

	// Determine status
	if len(issues) > 0 {
		result.Status = CheckStatusWarn
		result.Details = fmt.Sprintf("Found %d worktrees with %d missing directories", len(worktrees), len(issues))
		result.Suggestions = []string{
			"Run 'git worktree prune' to clean up missing worktrees",
			"Recreate missing worktrees if needed",
		}
	} else {
		result.Status = CheckStatusPass
		result.Details = fmt.Sprintf("All %d worktrees are healthy", len(worktrees))
	}

	// Add warnings if any
	if len(warnings) > 0 {
		result.Details += fmt.Sprintf(" (%d warnings)", len(warnings))
	}

	return result
}

// CheckEditors checks for available editors
func (c *Checker) CheckEditors() CheckResult {
	result := CheckResult{
		Name:        "Editors",
		Description: "Check for supported editors (Cursor, VS Code)",
	}

	var available []string
	var details []string

	// Check for Cursor
	if c.isEditorAvailable("cursor") {
		available = append(available, "Cursor")
		details = append(details, "✓ Cursor is available")
	} else {
		details = append(details, "✗ Cursor not found")
	}

	// Check for VS Code
	if c.isEditorAvailable("code") {
		available = append(available, "VS Code")
		details = append(details, "✓ VS Code is available")
	} else {
		details = append(details, "✗ VS Code not found")
	}

	// Determine status
	if len(available) == 0 {
		result.Status = CheckStatusWarn
		result.Details = "No supported editors found"
		result.Suggestions = []string{
			"Install Cursor from https://cursor.sh/",
			"Install VS Code from https://code.visualstudio.com/",
		}
	} else {
		result.Status = CheckStatusPass
		result.Details = fmt.Sprintf("Available editors: %s", strings.Join(available, ", "))
	}

	return result
}

// CheckConfiguration checks Hatcher configuration files
func (c *Checker) CheckConfiguration() CheckResult {
	result := CheckResult{
		Name:        "Configuration",
		Description: "Check Hatcher configuration files",
	}

	if c.repo == nil {
		result.Status = CheckStatusWarn
		result.Details = "No Git repository context for configuration check"
		return result
	}

	root, err := c.repo.GetRoot()
	if err != nil {
		result.Status = CheckStatusWarn
		result.Details = "Could not determine repository root"
		return result
	}

	var details []string
	var suggestions []string

	// Check for auto-copy configuration
	autoCopyPath := filepath.Join(root, ".hatcher-auto-copy.json")
	if _, err := os.Stat(autoCopyPath); err == nil {
		details = append(details, "✓ Auto-copy configuration found")
	} else {
		details = append(details, "✗ No auto-copy configuration")
		suggestions = append(suggestions, "Create .hatcher-auto-copy.json for automatic file copying")
	}

	// Check for global configuration
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalConfigPath := filepath.Join(homeDir, ".hatcher", "config.json")
		if _, err := os.Stat(globalConfigPath); err == nil {
			details = append(details, "✓ Global configuration found")
		} else {
			details = append(details, "✗ No global configuration")
		}
	}

	// Determine status
	if len(suggestions) == 0 {
		result.Status = CheckStatusPass
	} else {
		result.Status = CheckStatusWarn
	}

	result.Details = strings.Join(details, "\n")
	result.Suggestions = suggestions

	return result
}

// CheckPermissions checks file and directory permissions
func (c *Checker) CheckPermissions() CheckResult {
	result := CheckResult{
		Name:        "Permissions",
		Description: "Check file and directory permissions",
	}

	if c.repo == nil {
		result.Status = CheckStatusWarn
		result.Details = "No Git repository context for permission check"
		return result
	}

	root, err := c.repo.GetRoot()
	if err != nil {
		result.Status = CheckStatusWarn
		result.Details = "Could not determine repository root"
		return result
	}

	var issues []string

	// Check repository directory permissions
	if info, err := os.Stat(root); err != nil {
		issues = append(issues, "Cannot access repository directory")
	} else if !info.IsDir() {
		issues = append(issues, "Repository root is not a directory")
	}

	// Check write permissions in parent directory (for creating worktrees)
	parentDir := filepath.Dir(root)
	if info, err := os.Stat(parentDir); err != nil {
		issues = append(issues, "Cannot access parent directory")
	} else if !info.IsDir() {
		issues = append(issues, "Parent directory is not accessible")
	}

	// Determine status
	if len(issues) > 0 {
		result.Status = CheckStatusFail
		result.Details = strings.Join(issues, "; ")
		result.Suggestions = []string{
			"Check directory permissions",
			"Ensure you have read/write access to the repository and parent directory",
		}
	} else {
		result.Status = CheckStatusPass
		result.Details = "All required permissions are available"
	}

	return result
}

// isEditorAvailable checks if an editor command is available
func (c *Checker) isEditorAvailable(command string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// On macOS, check if the application exists
		switch command {
		case "cursor":
			cmd = exec.Command("mdfind", "kMDItemCFBundleIdentifier == 'com.todesktop.230313mzl4w4u92'")
		case "code":
			cmd = exec.Command("mdfind", "kMDItemCFBundleIdentifier == 'com.microsoft.VSCode'")
		}
	default:
		// On other platforms, check if command is in PATH
		cmd = exec.Command("which", command)
	}

	if cmd == nil {
		return false
	}

	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

// calculateSummary calculates the diagnostic summary
func (c *Checker) calculateSummary(checks []CheckResult) DiagnosticSummary {
	summary := DiagnosticSummary{
		Total: len(checks),
	}

	for _, check := range checks {
		switch check.Status {
		case CheckStatusPass:
			summary.Passed++
		case CheckStatusWarn:
			summary.Warned++
		case CheckStatusFail:
			summary.Failed++
		}
	}

	summary.Healthy = summary.Failed == 0

	return summary
}

// FormatAsTable formats the diagnostic result as a table
func (r *DiagnosticResult) FormatAsTable() string {
	var output bytes.Buffer
	w := tabwriter.NewWriter(&output, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(w, "CHECK\tSTATUS\tDETAILS")
	fmt.Fprintln(w, "-----\t------\t-------")

	// Rows
	for _, check := range r.Checks {
		var status string
		switch check.Status {
		case CheckStatusPass:
			status = "PASS"
		case CheckStatusWarn:
			status = "WARN"
		case CheckStatusFail:
			status = "FAIL"
		}

		// Truncate details for table display
		details := check.Details
		if len(details) > 60 {
			details = details[:57] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", check.Name, status, details)
	}

	w.Flush()

	// Add summary
	fmt.Fprintf(&output, "\nSummary: %d total, %d passed, %d warned, %d failed\n",
		r.Summary.Total, r.Summary.Passed, r.Summary.Warned, r.Summary.Failed)

	if r.Summary.Healthy {
		fmt.Fprintln(&output, "Overall Status: ✅ Healthy")
	} else {
		fmt.Fprintln(&output, "Overall Status: ❌ Issues Found")
	}

	return output.String()
}

// FormatAsJSON formats the diagnostic result as JSON
func (r *DiagnosticResult) FormatAsJSON() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal JSON: %s"}`, err.Error())
	}
	return string(data)
}

// FormatAsSimple formats the diagnostic result as a simple list
func (r *DiagnosticResult) FormatAsSimple() string {
	var output strings.Builder

	for _, check := range r.Checks {
		var icon string
		switch check.Status {
		case CheckStatusPass:
			icon = "✅"
		case CheckStatusWarn:
			icon = "⚠️"
		case CheckStatusFail:
			icon = "❌"
		}

		fmt.Fprintf(&output, "%s %s: %s\n", icon, check.Name, check.Details)

		// Add suggestions if any
		for _, suggestion := range check.Suggestions {
			fmt.Fprintf(&output, "   💡 %s\n", suggestion)
		}
	}

	// Add summary
	fmt.Fprintf(&output, "\n📊 Summary: %d total, %d passed, %d warned, %d failed\n",
		r.Summary.Total, r.Summary.Passed, r.Summary.Warned, r.Summary.Failed)

	return output.String()
}

// GetOverallStatus returns the overall status based on all checks
func (r *DiagnosticResult) GetOverallStatus() CheckStatus {
	if r.Summary.Failed > 0 {
		return CheckStatusFail
	}
	if r.Summary.Warned > 0 {
		return CheckStatusWarn
	}
	return CheckStatusPass
}
