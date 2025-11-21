package autocopy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoCopier_CopyFiles(t *testing.T) {
	// Create test repository with files to copy
	testRepo := helpers.NewTestGitRepository(t, "test-project")

	// Create test files and directories
	testRepo.CreateDirectory(".ai")
	testRepo.CreateFile(".ai/prompts.md", "# AI Prompts")
	testRepo.CreateFile(".ai/templates.md", "# Templates")
	testRepo.CreateFile(".cursorrules", "# Cursor rules")
	testRepo.CreateFile("CLAUDE.md", "# Claude context")
	testRepo.CreateFile("README.md", "# Project README")
	testRepo.CommitAll("Add test files")

	// Create destination directory
	dstDir := filepath.Join(testRepo.TempDir, "destination")
	err := os.MkdirAll(dstDir, 0755)
	require.NoError(t, err)

	t.Run("copy files with new format config", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:      ".ai/",
					Directory: boolPtr(true),
					Recursive: false,
					RootOnly:  true,
				},
				{
					Path:     ".cursorrules",
					RootOnly: true,
				},
				{
					Path:     "CLAUDE.md",
					RootOnly: true,
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Verify files were copied
		assert.Len(t, copiedFiles, 3)
		assert.Contains(t, copiedFiles, ".ai/")
		assert.Contains(t, copiedFiles, ".cursorrules")
		assert.Contains(t, copiedFiles, "CLAUDE.md")

		// Verify actual files exist
		assert.DirExists(t, filepath.Join(dstDir, ".ai"))
		assert.FileExists(t, filepath.Join(dstDir, ".ai", "prompts.md"))
		assert.FileExists(t, filepath.Join(dstDir, ".ai", "templates.md"))
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "CLAUDE.md"))

		// Verify content
		content, err := os.ReadFile(filepath.Join(dstDir, ".cursorrules"))
		require.NoError(t, err)
		assert.Equal(t, "# Cursor rules", string(content))
	})

	t.Run("copy files with legacy format config", func(t *testing.T) {
		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 0,
			Files:   []string{".ai/", ".cursorrules", "CLAUDE.md"},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Verify files were copied
		assert.Len(t, copiedFiles, 3)
		assert.DirExists(t, filepath.Join(dstDir, ".ai"))
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "CLAUDE.md"))
	})

	t.Run("copy with autoDetect", func(t *testing.T) {
		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:       ".cursorrules",
					AutoDetect: true,
					RootOnly:   true,
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should detect .cursorrules as a file and copy it
		assert.Len(t, copiedFiles, 1)
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
	})

	t.Run("copy with recursive option", func(t *testing.T) {
		// Create nested structure
		testRepo.CreateDirectory("src/components")
		testRepo.CreateFile("src/.cursorrules", "# Src cursor rules")
		testRepo.CreateFile("src/components/.cursorrules", "# Component cursor rules")
		testRepo.CommitAll("Add nested files")

		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:      ".cursorrules",
					Recursive: true,
					RootOnly:  false,
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should find and copy all .cursorrules files
		assert.Len(t, copiedFiles, 3) // root, src, src/components
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "components", ".cursorrules"))
	})

	t.Run("copy with rootOnly option", func(t *testing.T) {
		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:      ".cursorrules",
					Recursive: true,
					RootOnly:  true, // Only root level
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should only copy root level .cursorrules
		assert.Len(t, copiedFiles, 1)
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.NoFileExists(t, filepath.Join(dstDir, "src", ".cursorrules"))
	})

	t.Run("handle non-existent files", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:     "non-existent-file.txt",
					RootOnly: true,
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should not fail, but no files copied
		assert.Empty(t, copiedFiles)
	})

	t.Run("handle permission errors", func(t *testing.T) {
		// Create a read-only destination directory
		readOnlyDir := filepath.Join(testRepo.TempDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0444) // Read-only
		require.NoError(t, err)
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path:     ".cursorrules",
					RootOnly: true,
				},
			},
		}

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, readOnlyDir, config)

		// Should return error due to permission issues
		assert.Error(t, err)
		assert.Empty(t, copiedFiles)
	})
}

func TestAutoCopier_ProcessGlobPattern(t *testing.T) {
	// Create test repository with files
	testRepo := helpers.NewTestGitRepository(t, "glob-test")

	// Create test files matching glob patterns
	testRepo.CreateFile("config.json", `{"test": true}`)
	testRepo.CreateFile("package.json", `{"name": "test"}`)
	testRepo.CreateDirectory("src/config")
	testRepo.CreateFile("src/config/app.json", `{"app": true}`)
	testRepo.CreateFile("src/config/db.json", `{"db": true}`)
	testRepo.CreateFile("docs/config.yaml", `test: true`)
	testRepo.CommitAll("Add glob test files")

	// Create destination directory
	dstDir := filepath.Join(testRepo.TempDir, "destination")
	err := os.MkdirAll(dstDir, 0755)
	require.NoError(t, err)

	t.Run("simple glob pattern", func(t *testing.T) {
		copier := NewAutoCopier()
		copiedFiles, err := copier.ProcessGlobPattern("*.json", testRepo.RepoDir, dstDir)
		require.NoError(t, err)

		// Should match config.json and package.json
		assert.Len(t, copiedFiles, 2)
		assert.FileExists(t, filepath.Join(dstDir, "config.json"))
		assert.FileExists(t, filepath.Join(dstDir, "package.json"))
	})

	t.Run("recursive glob pattern", func(t *testing.T) {
		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		copier := NewAutoCopier()
		copiedFiles, err := copier.ProcessGlobPattern("**/*.json", testRepo.RepoDir, dstDir)
		require.NoError(t, err)

		// Should match all .json files recursively
		assert.Len(t, copiedFiles, 4) // config.json, package.json, src/config/app.json, src/config/db.json
		assert.FileExists(t, filepath.Join(dstDir, "config.json"))
		assert.FileExists(t, filepath.Join(dstDir, "package.json"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "config", "app.json"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "config", "db.json"))
	})

	t.Run("directory-specific glob pattern", func(t *testing.T) {
		// Clean destination
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		copier := NewAutoCopier()
		copiedFiles, err := copier.ProcessGlobPattern("src/config/*.json", testRepo.RepoDir, dstDir)
		require.NoError(t, err)

		// Should only match files in src/config/
		assert.Len(t, copiedFiles, 2) // src/config/app.json, src/config/db.json
		assert.FileExists(t, filepath.Join(dstDir, "src", "config", "app.json"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "config", "db.json"))
	})

	t.Run("no matches", func(t *testing.T) {
		copier := NewAutoCopier()
		copiedFiles, err := copier.ProcessGlobPattern("*.nonexistent", testRepo.RepoDir, dstDir)
		require.NoError(t, err)

		// Should return empty list, no error
		assert.Empty(t, copiedFiles)
	})
}

func TestAutoCopier_UpdateGitignore(t *testing.T) {
	testRepo := helpers.NewTestGitRepository(t, "gitignore-test")

	t.Run("create new gitignore", func(t *testing.T) {
		copier := NewAutoCopier()
		files := []string{".ai/", ".cursorrules", "CLAUDE.md"}

		err := copier.UpdateGitignore(testRepo.RepoDir, files)
		require.NoError(t, err)

		// Verify .gitignore was created
		gitignorePath := filepath.Join(testRepo.RepoDir, ".gitignore")
		assert.FileExists(t, gitignorePath)

		// Verify content
		content, err := os.ReadFile(gitignorePath)
		require.NoError(t, err)

		gitignoreContent := string(content)
		assert.Contains(t, gitignoreContent, "# Auto-copied files (added by hatcher)")
		assert.Contains(t, gitignoreContent, ".ai/")
		assert.Contains(t, gitignoreContent, ".cursorrules")
		assert.Contains(t, gitignoreContent, "CLAUDE.md")
	})

	t.Run("append to existing gitignore", func(t *testing.T) {
		// Create existing .gitignore
		gitignorePath := filepath.Join(testRepo.RepoDir, ".gitignore")
		existingContent := "# Existing content\n*.log\nnode_modules/\n"
		err := os.WriteFile(gitignorePath, []byte(existingContent), 0644)
		require.NoError(t, err)

		copier := NewAutoCopier()
		files := []string{"new-file.txt"}

		err = copier.UpdateGitignore(testRepo.RepoDir, files)
		require.NoError(t, err)

		// Verify content was appended
		content, err := os.ReadFile(gitignorePath)
		require.NoError(t, err)

		gitignoreContent := string(content)
		assert.Contains(t, gitignoreContent, "# Existing content")
		assert.Contains(t, gitignoreContent, "*.log")
		assert.Contains(t, gitignoreContent, "# Auto-copied files (added by hatcher)")
		assert.Contains(t, gitignoreContent, "new-file.txt")
	})

	t.Run("empty files list", func(t *testing.T) {
		copier := NewAutoCopier()
		err := copier.UpdateGitignore(testRepo.RepoDir, []string{})
		require.NoError(t, err)

		// Should not create or modify .gitignore
		gitignorePath := filepath.Join(testRepo.RepoDir, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			// If .gitignore exists, it should not be modified
			content, err := os.ReadFile(gitignorePath)
			require.NoError(t, err)

			// Should not contain new auto-copied section
			gitignoreContent := string(content)
			assert.NotContains(t, gitignoreContent, "# Auto-copied files (added by hatcher)")
		}
	})
}
