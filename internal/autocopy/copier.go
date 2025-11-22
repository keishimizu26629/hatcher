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

// NewLegacyAutoCopier creates a new AutoCopier instance with legacy interface
// This is for backward compatibility with existing tests
func NewLegacyAutoCopier() *LegacyAutoCopier {
	return &LegacyAutoCopier{}
}

// LegacyAutoCopier provides backward compatibility
type LegacyAutoCopier struct{}

// CopyFiles provides legacy interface for file copying
func (lac *LegacyAutoCopier) CopyFiles(sourceDir, destDir string, config *AutoCopyConfig) ([]string, error) {
	if config == nil {
		return []string{}, nil
	}

	var copiedFiles []string

	// Handle legacy format
	if config.Version == 0 && len(config.Files) > 0 {
		for _, file := range config.Files {
			copied, err := lac.copySinglePath(sourceDir, destDir, file)
			if err != nil {
				return nil, err
			}
			if copied {
				copiedFiles = append(copiedFiles, file)
			}
		}
		return copiedFiles, nil
	}

	// Handle new format
	for _, item := range config.Items {
		if item.IsGlobPattern() || (item.Recursive && !item.RootOnly) {
			// Use glob pattern processing for recursive searches
			pattern := item.Path
			if item.Recursive && !item.IsGlobPattern() && !item.RootOnly {
				// Convert to recursive glob pattern
				pattern = "**/" + item.Path
			}
			files, err := lac.ProcessGlobPatternWithOptions(pattern, sourceDir, destDir, item)
			if err != nil {
				return nil, err
			}
			copiedFiles = append(copiedFiles, files...)
		} else {
			copied, err := lac.copySingleItem(sourceDir, destDir, item)
			if err != nil {
				return nil, err
			}
			copiedFiles = append(copiedFiles, copied...)
		}
	}

	return copiedFiles, nil
}

// ProcessGlobPatternWithOptions provides glob processing with item options
func (lac *LegacyAutoCopier) ProcessGlobPatternWithOptions(pattern, sourceDir, destDir string, item AutoCopyItem) ([]string, error) {
	// Handle recursive patterns (starting with **/)
	if strings.HasPrefix(pattern, "**/") {
		filename := strings.TrimPrefix(pattern, "**/")
		return lac.findRecursiveFilesWithRootOnly(filename, sourceDir, destDir, item.RootOnly)
	}

	// Use regular glob processing
	return lac.ProcessGlobPattern(pattern, sourceDir, destDir)
}

// ProcessGlobPattern provides legacy interface for glob processing
func (lac *LegacyAutoCopier) ProcessGlobPattern(pattern, sourceDir, destDir string) ([]string, error) {
	var copiedFiles []string

	// Handle recursive patterns (starting with **/)
	if strings.HasPrefix(pattern, "**/") {
		filename := strings.TrimPrefix(pattern, "**/")
		return lac.findRecursiveFiles(filename, sourceDir, destDir)
	}

	// Use filepath.Glob to find matching files
	searchPattern := filepath.Join(sourceDir, pattern)
	matches, err := filepath.Glob(searchPattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern error: %w", err)
	}

	for _, match := range matches {
		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, match)
		if err != nil {
			continue
		}

		destPath := filepath.Join(destDir, relPath)

		// Check if it's a file or directory
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.IsDir() {
			err = lac.copyDirectory(match, destPath, true)
		} else {
			err = lac.copyFile(match, destPath)
		}

		if err == nil {
			copiedFiles = append(copiedFiles, relPath)
		}
	}

	return copiedFiles, nil
}

// findRecursiveFiles finds files recursively using filepath.Walk
func (lac *LegacyAutoCopier) findRecursiveFiles(filename, sourceDir, destDir string) ([]string, error) {
	var copiedFiles []string

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if filename matches
		if filepath.Base(path) == filename {
			// Get relative path from source directory
			relPath, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}

			destPath := filepath.Join(destDir, relPath)

			// Copy the file
			if err := lac.copyFile(path, destPath); err != nil {
				return err
			}

			copiedFiles = append(copiedFiles, relPath)
		}

		return nil
	})

	return copiedFiles, err
}

// findRecursiveFilesWithRootOnly finds files recursively with rootOnly option
func (lac *LegacyAutoCopier) findRecursiveFilesWithRootOnly(filename, sourceDir, destDir string, rootOnly bool) ([]string, error) {
	var copiedFiles []string

	if rootOnly {
		// Only check root level
		rootPath := filepath.Join(sourceDir, filename)
		if info, err := os.Stat(rootPath); err == nil && !info.IsDir() {
			destPath := filepath.Join(destDir, filename)
			if err := lac.copyFile(rootPath, destPath); err != nil {
				return nil, err
			}
			copiedFiles = append(copiedFiles, filename)
		}
		return copiedFiles, nil
	}

	// Use regular recursive search
	return lac.findRecursiveFiles(filename, sourceDir, destDir)
}

// UpdateGitignore provides legacy interface for gitignore updates
func (lac *LegacyAutoCopier) UpdateGitignore(repoDir string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(repoDir, ".gitignore")

	// Read existing .gitignore content
	var existingContent string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = string(data)
	}

	// Check if we already have our section
	if strings.Contains(existingContent, "# Auto-copied files (added by hatcher)") {
		return nil // Already updated
	}

	// Prepare new content to append
	var newContent strings.Builder
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		newContent.WriteString("\n")
	}
	newContent.WriteString("\n# Auto-copied files (added by hatcher)\n")
	for _, file := range files {
		newContent.WriteString(file + "\n")
	}

	// Append to .gitignore
	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(newContent.String())
	return err
}

// copySinglePath copies a single file or directory path
func (lac *LegacyAutoCopier) copySinglePath(sourceDir, destDir, path string) (bool, error) {
	sourcePath := filepath.Join(sourceDir, path)
	destPath := filepath.Join(destDir, path)

	// Check if source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Skip non-existent files
		}
		return false, err
	}

	if info.IsDir() {
		return true, lac.copyDirectory(sourcePath, destPath, false)
	} else {
		return true, lac.copyFile(sourcePath, destPath)
	}
}

// copySingleItem copies a single AutoCopyItem
func (lac *LegacyAutoCopier) copySingleItem(sourceDir, destDir string, item AutoCopyItem) ([]string, error) {
	sourcePath := filepath.Join(sourceDir, item.Path)
	destPath := filepath.Join(destDir, item.Path)

	// Check if source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) && item.AutoDetect {
			return []string{}, nil // Skip non-existent files when auto-detecting
		}
		if os.IsNotExist(err) {
			return []string{}, nil // Skip non-existent files
		}
		return nil, err
	}

	if info.IsDir() {
		if item.Directory != nil && !*item.Directory {
			return nil, fmt.Errorf("expected file but found directory: %s", sourcePath)
		}
		// For directories, always copy contents unless explicitly set to false
		recursive := item.Recursive
		if item.Directory != nil && *item.Directory {
			recursive = true // Default to recursive for explicitly marked directories
		}
		err = lac.copyDirectory(sourcePath, destPath, recursive)
		if err != nil {
			return nil, err
		}
		return []string{item.Path}, nil
	} else {
		if item.Directory != nil && *item.Directory {
			return nil, fmt.Errorf("expected directory but found file: %s", sourcePath)
		}
		err = lac.copyFile(sourcePath, destPath)
		if err != nil {
			return nil, err
		}
		return []string{item.Path}, nil
	}
}

// copyFile copies a single file
func (lac *LegacyAutoCopier) copyFile(sourcePath, destPath string) error {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}
	defer destFile.Close()

	// Copy content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Copy permissions
	sourceInfo, err := os.Stat(sourcePath)
	if err == nil {
		os.Chmod(destPath, sourceInfo.Mode())
	}

	return nil
}

// copyDirectory copies a directory and optionally its contents
func (lac *LegacyAutoCopier) copyDirectory(sourcePath, destPath string, recursive bool) error {
	// Create destination directory
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destPath, err)
	}

	if !recursive {
		return nil // Only create the directory structure, not contents
	}

	// Copy directory contents recursively
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Get relative path from source
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		destItemPath := filepath.Join(destPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destItemPath, info.Mode())
		} else {
			return lac.copyFile(path, destItemPath)
		}
	})
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
	// Use legacy copier for sequential processing
	legacyCopier := NewLegacyAutoCopier()
	copiedFiles, err := legacyCopier.CopyFiles(sourceDir, destDir, ac.config)
	if err != nil {
		return err
	}

	// Update .gitignore if we copied any files
	if len(copiedFiles) > 0 && !ac.options.NoGitignoreUpdate {
		if err := legacyCopier.UpdateGitignore(destDir, copiedFiles); err != nil {
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
