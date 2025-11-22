package worktree

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateWorktreePath generates the full path for a worktree
func GenerateWorktreePath(repoRoot, projectName, branchName string) string {
	branchNameSafe := SanitizeBranchName(branchName)
	dirName := fmt.Sprintf("%s-%s", projectName, branchNameSafe)
	parentDir := filepath.Dir(repoRoot)
	return filepath.Join(parentDir, dirName)
}

// IsHatcherWorktree checks if a worktree was created by Hatcher based on naming convention
func IsHatcherWorktree(worktreePath, projectName string) bool {
	dirName := filepath.Base(worktreePath)
	expectedPrefix := projectName + "-"
	return strings.HasPrefix(dirName, expectedPrefix)
}

// SanitizeBranchName converts a branch name to a filesystem-safe format
func SanitizeBranchName(branch string) string {
	// Replace / with -
	safe := strings.ReplaceAll(branch, "/", "-")

	// Replace other problematic characters
	replacements := map[string]string{
		" ":  "-",
		"@":  "-",
		"#":  "-",
		":":  "-",
		"*":  "-",
		"?":  "-",
		"\"": "-",
		"<":  "-",
		">":  "-",
		"|":  "-",
	}

	for old, new := range replacements {
		safe = strings.ReplaceAll(safe, old, new)
	}

	// Remove leading/trailing dashes
	safe = strings.Trim(safe, "-")

	// Collapse multiple consecutive dashes
	for strings.Contains(safe, "--") {
		safe = strings.ReplaceAll(safe, "--", "-")
	}

	return safe
}
