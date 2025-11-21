package autocopy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// AutoCopierOptions contains options for the AutoCopier
type AutoCopierOptions struct {
	NoGitignoreUpdate bool // Skip updating .gitignore
	UseParallel       bool // Use parallel processing
	MaxWorkers        int  // Maximum number of worker goroutines
	BufferSize        int  // Buffer size for file copying
	ShowProgress      bool // Show progress updates
	VerifyIntegrity   bool // Verify file integrity after copying
}

// AutoCopier handles automatic file copying operations
type AutoCopier struct {
	repo    git.Repository
	config  *AutoCopyConfig
	options AutoCopierOptions
}

// NewAutoCopier creates a new AutoCopier instance
func NewAutoCopier(repo git.Repository, config *AutoCopyConfig, options AutoCopierOptions) *AutoCopier {
	// Set default options
	if options.MaxWorkers <= 0 {
		options.MaxWorkers = 4
	}
	if options.BufferSize <= 0 {
		options.BufferSize = 64 * 1024 // 64KB
	}

	return &AutoCopier{
		repo:    repo,
		config:  config,
		options: options,
	}
}

// Run executes the auto-copy operation
func (ac *AutoCopier) Run(sourceDir, destDir string) error {
	if ac.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Use parallel copier if enabled
	if ac.options.UseParallel {
		return ac.runParallel(sourceDir, destDir)
	}

	// Use sequential copier (original implementation)
	return ac.runSequential(sourceDir, destDir)
}

// runParallel executes the auto-copy operation using parallel processing
func (ac *AutoCopier) runParallel(sourceDir, destDir string) error {
	parallelOptions := ParallelCopyOptions{
		MaxWorkers:      ac.options.MaxWorkers,
		BufferSize:      ac.options.BufferSize,
		ShowProgress:    ac.options.ShowProgress,
		VerifyIntegrity: ac.options.VerifyIntegrity,
		ContinueOnError: true, // Continue on individual file errors
	}

	// Set up progress callback if needed
	if ac.options.ShowProgress {
		parallelOptions.ProgressCallback = func(update ProgressUpdate) {
			switch update.Type {
			case ProgressTypeStart:
				fmt.Printf("🚀 %s\n", update.Message)
			case ProgressTypeProgress:
				fmt.Printf("📋 %s (%.1f%%)\n", update.Message, update.Percentage)
			case ProgressTypeComplete:
				fmt.Printf("✅ %s in %v\n", update.Message, update.ElapsedTime)
			}
		}
	}

	// Track copied files for .gitignore update
	var copiedFiles []string
	var copiedFilesMutex sync.Mutex

	parallelOptions.ErrorCallback = func(err CopyError) {
		fmt.Printf("⚠️  Failed to copy %s: %v\n", err.SourcePath, err.Error)
	}

	// Create parallel copier
	copier := NewParallelCopier(ac.repo, ac.config, parallelOptions)

	// Execute parallel copy
	if err := copier.Run(sourceDir, destDir); err != nil {
		return fmt.Errorf("parallel copy failed: %w", err)
	}

	// Collect copied files for .gitignore update
	// This is a simplified approach - in a real implementation,
	// you'd want to track this during the copy operation
	for _, item := range ac.config.Items {
		files, err := ac.findCopiedFiles(destDir, item)
		if err != nil {
			continue // Continue on error
		}
		copiedFilesMutex.Lock()
		copiedFiles = append(copiedFiles, files...)
		copiedFilesMutex.Unlock()
	}

	// Update .gitignore if we copied any files
	if len(copiedFiles) > 0 && !ac.options.NoGitignoreUpdate {
		if err := ac.repo.UpdateGitignore(copiedFiles); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	return nil
}

// runSequential executes the auto-copy operation sequentially (original implementation)
func (ac *AutoCopier) runSequential(sourceDir, destDir string) error {
	copiedFiles, err := ac.CopyFiles(sourceDir, destDir, ac.config)
	if err != nil {
		return err
	}

	// Update .gitignore if we copied any files
	if len(copiedFiles) > 0 && !ac.options.NoGitignoreUpdate {
		if err := ac.repo.UpdateGitignore(copiedFiles); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	return nil
}

// findCopiedFiles finds files that were copied for a given item
func (ac *AutoCopier) findCopiedFiles(destDir string, item AutoCopyItem) ([]string, error) {
	var files []string
	destPath := filepath.Join(destDir, item.Path)

	// Check if destination exists
	info, err := os.Stat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil // No files copied
		}
		return files, err
	}

	if info.IsDir() {
		// For directories, add the directory itself
		files = append(files, item.Path)
	} else {
		// For files, add the file
		files = append(files, item.Path)
	}

	return files, nil
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
