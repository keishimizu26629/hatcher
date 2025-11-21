package autocopy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AutoCopier handles automatic file copying operations
type AutoCopier struct {
	// Future: add configuration options like parallel workers, etc.
}

// NewAutoCopier creates a new AutoCopier instance
func NewAutoCopier() *AutoCopier {
	return &AutoCopier{}
}

// CopyFiles copies files according to the configuration
func (c *AutoCopier) CopyFiles(srcRoot, dstRoot string, config *AutoCopyConfig) ([]string, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	var copiedFiles []string

	// Handle legacy format
	if config.Version == 0 && len(config.Files) > 0 {
		for _, file := range config.Files {
			copied, err := c.copyLegacyFile(srcRoot, dstRoot, file)
			if err != nil {
				return copiedFiles, err
			}
			if copied {
				copiedFiles = append(copiedFiles, file)
			}
		}
		return copiedFiles, nil
	}

	// Handle new format
	for _, item := range config.Items {
		copied, err := c.copyItem(srcRoot, dstRoot, item)
		if err != nil {
			return copiedFiles, err
		}
		copiedFiles = append(copiedFiles, copied...)
	}

	return copiedFiles, nil
}

// copyLegacyFile copies a file using legacy format rules
func (c *AutoCopier) copyLegacyFile(srcRoot, dstRoot, file string) (bool, error) {
	srcPath := filepath.Join(srcRoot, file)
	dstPath := filepath.Join(dstRoot, file)

	// Check if source exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist, skip silently
		}
		return false, err
	}

	if srcInfo.IsDir() {
		return c.copyDirectory(srcPath, dstPath, false) // Non-recursive for legacy
	} else {
		return c.copyFile(srcPath, dstPath)
	}
}

// copyItem copies a file or directory according to the item configuration
func (c *AutoCopier) copyItem(srcRoot, dstRoot string, item AutoCopyItem) ([]string, error) {
	var copiedFiles []string

	// Handle glob patterns
	if item.IsGlobPattern() {
		return c.ProcessGlobPattern(item.Path, srcRoot, dstRoot)
	}

	// Handle single file/directory
	if item.RootOnly {
		// Only check root level
		copied, err := c.copySingleItem(srcRoot, dstRoot, item, "")
		if err != nil {
			return nil, err
		}
		if len(copied) > 0 {
			copiedFiles = append(copiedFiles, copied...)
		}
	} else if item.Recursive {
		// Search recursively
		err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcRoot, path)
			if err != nil {
				return err
			}

			// Skip root directory itself
			if relPath == "." {
				return nil
			}

			// Check if this path matches the item
			if c.pathMatches(relPath, item, info) {
				copied, err := c.copySingleItem(srcRoot, dstRoot, item, relPath)
				if err != nil {
					return err
				}
				copiedFiles = append(copiedFiles, copied...)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Non-recursive, root only
		copied, err := c.copySingleItem(srcRoot, dstRoot, item, "")
		if err != nil {
			return nil, err
		}
		copiedFiles = append(copiedFiles, copied...)
	}

	return copiedFiles, nil
}

// copySingleItem copies a single item (file or directory)
func (c *AutoCopier) copySingleItem(srcRoot, dstRoot string, item AutoCopyItem, relPath string) ([]string, error) {
	var itemPath string
	if relPath == "" {
		itemPath = item.Path
	} else {
		// For recursive search, use the found path
		if filepath.Base(relPath) == filepath.Base(item.Path) {
			itemPath = relPath
		} else {
			return nil, nil // Doesn't match
		}
	}

	srcPath := filepath.Join(srcRoot, itemPath)
	dstPath := filepath.Join(dstRoot, itemPath)

	// Check if source exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, skip silently
		}
		return nil, err
	}

	// Determine if it's a directory
	isDir := srcInfo.IsDir()
	if item.AutoDetect {
		// Use actual file system information
		isDir = srcInfo.IsDir()
	} else {
		// Use configuration or auto-detect from path
		isDir = item.IsDirectory()

		// Validate that the configuration matches reality
		if isDir != srcInfo.IsDir() {
			return nil, fmt.Errorf("path %s: configuration mismatch (expected dir=%v, actual dir=%v)",
				itemPath, isDir, srcInfo.IsDir())
		}
	}

	if isDir {
		copied, err := c.copyDirectory(srcPath, dstPath, item.Recursive)
		if err != nil {
			return nil, err
		}
		if copied {
			return []string{itemPath}, nil
		}
	} else {
		copied, err := c.copyFile(srcPath, dstPath)
		if err != nil {
			return nil, err
		}
		if copied {
			return []string{itemPath}, nil
		}
	}

	return nil, nil
}

// pathMatches checks if a path matches the item criteria
func (c *AutoCopier) pathMatches(relPath string, item AutoCopyItem, info os.FileInfo) bool {
	// Simple name matching for now
	return filepath.Base(relPath) == filepath.Base(item.Path)
}

// ProcessGlobPattern processes a glob pattern and copies matching files
func (c *AutoCopier) ProcessGlobPattern(pattern, srcRoot, dstRoot string) ([]string, error) {
	var copiedFiles []string

	// Use filepath.Glob for pattern matching
	globPattern := filepath.Join(srcRoot, pattern)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
	}

	for _, match := range matches {
		relPath, err := filepath.Rel(srcRoot, match)
		if err != nil {
			continue
		}

		srcPath := match
		dstPath := filepath.Join(dstRoot, relPath)

		// Check if it's a file or directory
		info, err := os.Stat(srcPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			copied, err := c.copyDirectory(srcPath, dstPath, false)
			if err != nil {
				return copiedFiles, err
			}
			if copied {
				copiedFiles = append(copiedFiles, relPath)
			}
		} else {
			copied, err := c.copyFile(srcPath, dstPath)
			if err != nil {
				return copiedFiles, err
			}
			if copied {
				copiedFiles = append(copiedFiles, relPath)
			}
		}
	}

	return copiedFiles, nil
}

// copyFile copies a single file
func (c *AutoCopier) copyFile(srcPath, dstPath string) (bool, error) {
	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create destination directory %s: %w", dstDir, err)
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return false, fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return false, fmt.Errorf("failed to create destination file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return false, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Copy permissions
	srcInfo, err := os.Stat(srcPath)
	if err == nil {
		os.Chmod(dstPath, srcInfo.Mode())
	}

	return true, nil
}

// copyDirectory copies a directory and optionally its contents
func (c *AutoCopier) copyDirectory(srcPath, dstPath string, recursive bool) (bool, error) {
	// Create destination directory
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return false, fmt.Errorf("failed to create destination directory %s: %w", dstPath, err)
	}

	if !recursive {
		// Only copy the directory structure, not contents
		return true, nil
	}

	// Copy directory contents recursively
	return c.copyDirectoryRecursive(srcPath, dstPath)
}

// copyDirectoryRecursive copies directory contents recursively
func (c *AutoCopier) copyDirectoryRecursive(srcPath, dstPath string) (bool, error) {
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return false, fmt.Errorf("failed to read directory %s: %w", srcPath, err)
	}

	for _, entry := range entries {
		srcEntryPath := filepath.Join(srcPath, entry.Name())
		dstEntryPath := filepath.Join(dstPath, entry.Name())

		if entry.IsDir() {
			_, err := c.copyDirectory(srcEntryPath, dstEntryPath, true)
			if err != nil {
				return false, err
			}
		} else {
			_, err := c.copyFile(srcEntryPath, dstEntryPath)
			if err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// UpdateGitignore adds files to .gitignore
func (c *AutoCopier) UpdateGitignore(repoRoot string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing .gitignore
	var existing []byte
	if _, err := os.Stat(gitignorePath); err == nil {
		existing, _ = os.ReadFile(gitignorePath)
	}

	// Prepare new content
	content := string(existing)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Add separator comment
	content += "\n# Auto-copied files (added by hatcher)\n"

	// Add files
	for _, file := range files {
		content += file + "\n"
	}

	// Write back to .gitignore
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}
