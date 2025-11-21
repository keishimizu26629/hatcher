package autocopy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoCopyIntegration(t *testing.T) {
	// Create test repository with realistic file structure
	testRepo := helpers.NewTestGitRepository(t, "integration-test")

	// Create AI and development files
	testRepo.CreateDirectory(".ai")
	testRepo.CreateFile(".ai/prompts.md", "# AI Prompts\n\n## Code Generation\n...")
	testRepo.CreateFile(".ai/templates.md", "# Templates\n\n## React Component\n...")
	testRepo.CreateFile(".cursorrules", "# Cursor Rules\n\n- Use TypeScript\n- Follow ESLint rules")
	testRepo.CreateFile("CLAUDE.md", "# Claude Context\n\nThis is a Git worktree management tool...")
	testRepo.CreateFile("README.md", "# Project README")
	testRepo.CreateFile("package.json", `{"name": "test-project"}`)

	// Create nested structure with rules
	testRepo.CreateDirectory("src/components")
	testRepo.CreateFile("src/.cursorrules", "# Src specific rules")
	testRepo.CreateFile("src/components/.cursorrules", "# Component specific rules")

	testRepo.CommitAll("Add development files")

	t.Run("full workflow with project-specific config", func(t *testing.T) {
		// Create project-specific configuration
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		configContent := `{
			"version": 1,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": true,
					"rootOnly": true
				},
				{
					"path": ".cursorrules",
					"autoDetect": true,
					"recursive": true,
					"rootOnly": false
				},
				{
					"path": "CLAUDE.md",
					"directory": false,
					"rootOnly": true
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err = os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Create destination worktree directory
		worktreePath := filepath.Join(testRepo.TempDir, "integration-test-feature-autocopy")
		err = os.MkdirAll(worktreePath, 0755)
		require.NoError(t, err)

		// Load and validate configuration
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)
		assert.Equal(t, 1, config.Version)
		assert.Len(t, config.Items, 3)

		err = ValidateAutoCopyConfig(config)
		require.NoError(t, err)

		// Execute auto-copy
		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, worktreePath, config)
		require.NoError(t, err)

		// Verify copied files
		expectedFiles := []string{".ai/", ".cursorrules", "src/.cursorrules", "src/components/.cursorrules", "CLAUDE.md"}
		assert.Len(t, copiedFiles, len(expectedFiles))

		for _, expectedFile := range expectedFiles {
			assert.Contains(t, copiedFiles, expectedFile)
		}

		// Verify actual file existence and content
		assert.DirExists(t, filepath.Join(worktreePath, ".ai"))
		assert.FileExists(t, filepath.Join(worktreePath, ".ai", "prompts.md"))
		assert.FileExists(t, filepath.Join(worktreePath, ".ai", "templates.md"))
		assert.FileExists(t, filepath.Join(worktreePath, ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "src", ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "src", "components", ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "CLAUDE.md"))

		// Verify content integrity
		originalContent, err := os.ReadFile(filepath.Join(testRepo.RepoDir, ".cursorrules"))
		require.NoError(t, err)
		copiedContent, err := os.ReadFile(filepath.Join(worktreePath, ".cursorrules"))
		require.NoError(t, err)
		assert.Equal(t, originalContent, copiedContent)

		// Update .gitignore
		err = copier.UpdateGitignore(worktreePath, copiedFiles)
		require.NoError(t, err)

		// Verify .gitignore content
		gitignorePath := filepath.Join(worktreePath, ".gitignore")
		assert.FileExists(t, gitignorePath)

		gitignoreContent, err := os.ReadFile(gitignorePath)
		require.NoError(t, err)

		gitignoreStr := string(gitignoreContent)
		assert.Contains(t, gitignoreStr, "# Auto-copied files (added by hatcher)")
		for _, file := range copiedFiles {
			assert.Contains(t, gitignoreStr, file)
		}
	})

	t.Run("workflow with glob patterns", func(t *testing.T) {
		// Create configuration with glob patterns
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"version": 1,
			"items": [
				{
					"path": "**/.cursorrules"
				},
				{
					"path": "*.md"
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Create destination worktree directory
		worktreePath := filepath.Join(testRepo.TempDir, "integration-test-glob-pattern")
		err = os.MkdirAll(worktreePath, 0755)
		require.NoError(t, err)

		// Execute auto-copy
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, worktreePath, config)
		require.NoError(t, err)

		// Should find all .cursorrules files and .md files in root
		expectedMinFiles := 5 // 3 .cursorrules + 2 .md files (README.md, CLAUDE.md)
		assert.GreaterOrEqual(t, len(copiedFiles), expectedMinFiles)

		// Verify specific files
		assert.FileExists(t, filepath.Join(worktreePath, ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "src", ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "src", "components", ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "README.md"))
		assert.FileExists(t, filepath.Join(worktreePath, "CLAUDE.md"))
	})

	t.Run("workflow with legacy format", func(t *testing.T) {
		// Create legacy format configuration
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"files": [
				".ai/",
				".cursorrules",
				"CLAUDE.md"
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Create destination worktree directory
		worktreePath := filepath.Join(testRepo.TempDir, "integration-test-legacy")
		err = os.MkdirAll(worktreePath, 0755)
		require.NoError(t, err)

		// Execute auto-copy
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)
		assert.Equal(t, 0, config.Version) // Legacy format
		assert.Len(t, config.Files, 3)

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, worktreePath, config)
		require.NoError(t, err)

		// Should copy the specified files
		assert.Len(t, copiedFiles, 3)
		assert.Contains(t, copiedFiles, ".ai/")
		assert.Contains(t, copiedFiles, ".cursorrules")
		assert.Contains(t, copiedFiles, "CLAUDE.md")

		// Verify files exist
		assert.DirExists(t, filepath.Join(worktreePath, ".ai"))
		assert.FileExists(t, filepath.Join(worktreePath, ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "CLAUDE.md"))
	})

	t.Run("error handling - invalid configuration", func(t *testing.T) {
		// Create invalid configuration
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"version": 1,
			"items": [
				{
					"path": "../dangerous/path",
					"rootOnly": true
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Load configuration should succeed
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)

		// But validation should fail
		err = ValidateAutoCopyConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous path")
	})

	t.Run("performance with many files", func(t *testing.T) {
		// Create many files for performance testing
		for i := 0; i < 50; i++ {
			dir := filepath.Join("perf", fmt.Sprintf("dir%d", i))
			testRepo.CreateDirectory(dir)
			testRepo.CreateFile(filepath.Join(dir, "config.json"), fmt.Sprintf(`{"id": %d}`, i))
			testRepo.CreateFile(filepath.Join(dir, ".cursorrules"), fmt.Sprintf("# Rules for dir%d", i))
		}
		testRepo.CommitAll("Add performance test files")

		// Create configuration to copy all files
		configDir := filepath.Join(testRepo.RepoDir, ".worktree-files")
		configContent := `{
			"version": 1,
			"items": [
				{
					"path": "**/config.json"
				},
				{
					"path": "**/.cursorrules"
				}
			]
		}`

		configFile := filepath.Join(configDir, "auto-copy-files.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Create destination
		worktreePath := filepath.Join(testRepo.TempDir, "integration-test-performance")
		err = os.MkdirAll(worktreePath, 0755)
		require.NoError(t, err)

		// Execute auto-copy
		config, err := LoadAutoCopyConfig([]string{configFile})
		require.NoError(t, err)

		copier := NewAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, worktreePath, config)
		require.NoError(t, err)

		// Should copy many files efficiently
		expectedFiles := 50*2 + 3 // 50 dirs * 2 files + original 3 .cursorrules
		assert.Len(t, copiedFiles, expectedFiles)

		// Verify some files exist
		assert.FileExists(t, filepath.Join(worktreePath, "perf", "dir0", "config.json"))
		assert.FileExists(t, filepath.Join(worktreePath, "perf", "dir25", ".cursorrules"))
		assert.FileExists(t, filepath.Join(worktreePath, "perf", "dir49", "config.json"))
	})
}
