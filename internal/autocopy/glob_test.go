package autocopy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobPatternMatching(t *testing.T) {
	// Create test repository with complex file structure
	testRepo := testutil.NewTestGitRepository(t, "glob-complex-test")

	// Create complex directory structure
	testRepo.CreateDirectory("src/components")
	testRepo.CreateDirectory("src/utils")
	testRepo.CreateDirectory("tests/unit")
	testRepo.CreateDirectory("tests/integration")
	testRepo.CreateDirectory("docs/api")

	// Create various files
	testRepo.CreateFile(".cursorrules", "# Root cursor rules")
	testRepo.CreateFile("src/.cursorrules", "# Src cursor rules")
	testRepo.CreateFile("src/components/.cursorrules", "# Components cursor rules")
	testRepo.CreateFile("tests/.cursorrules", "# Tests cursor rules")

	testRepo.CreateFile("config.json", `{"root": true}`)
	testRepo.CreateFile("src/config.json", `{"src": true}`)
	testRepo.CreateFile("tests/config.json", `{"tests": true}`)

	testRepo.CreateFile("rules.json", `{"type": "rules"}`)
	testRepo.CreateFile("src/utils/rules.json", `{"type": "utils-rules"}`)
	testRepo.CreateFile("docs/api/rules.json", `{"type": "api-rules"}`)

	testRepo.CommitAll("Add complex file structure")

	// Create destination directory
	dstDir := filepath.Join(testRepo.TempDir, "destination")

	t.Run("double asterisk recursive pattern", func(t *testing.T) {
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "**/.cursorrules",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should find all .cursorrules files recursively
		assert.Len(t, copiedFiles, 4)
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "components", ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "tests", ".cursorrules"))
	})

	t.Run("specific directory pattern", func(t *testing.T) {
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "src/**/*.json",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should find JSON files only in src/ directory
		assert.Len(t, copiedFiles, 2) // src/config.json, src/utils/rules.json
		assert.FileExists(t, filepath.Join(dstDir, "src", "config.json"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "utils", "rules.json"))
		assert.NoFileExists(t, filepath.Join(dstDir, "config.json"))          // Root level excluded
		assert.NoFileExists(t, filepath.Join(dstDir, "tests", "config.json")) // Tests excluded
	})

	t.Run("multiple patterns in single config", func(t *testing.T) {
		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "**/rules.json",
				},
				{
					Path: "**/.cursorrules",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should find both rules.json and .cursorrules files
		assert.Len(t, copiedFiles, 7) // 3 rules.json + 4 .cursorrules

		// Verify rules.json files
		assert.FileExists(t, filepath.Join(dstDir, "rules.json"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "utils", "rules.json"))
		assert.FileExists(t, filepath.Join(dstDir, "docs", "api", "rules.json"))

		// Verify .cursorrules files
		assert.FileExists(t, filepath.Join(dstDir, ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "src", "components", ".cursorrules"))
		assert.FileExists(t, filepath.Join(dstDir, "tests", ".cursorrules"))
	})

	t.Run("pattern with question mark wildcard", func(t *testing.T) {
		// Create files with single character variations
		testRepo.CreateFile("file1.txt", "content1")
		testRepo.CreateFile("file2.txt", "content2")
		testRepo.CreateFile("file10.txt", "content10") // Should not match file?.txt
		testRepo.CreateFile("filea.txt", "contenta")
		testRepo.CommitAll("Add wildcard test files")

		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "file?.txt",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should match file1.txt, file2.txt, filea.txt but not file10.txt
		assert.Len(t, copiedFiles, 3)
		assert.FileExists(t, filepath.Join(dstDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(dstDir, "file2.txt"))
		assert.FileExists(t, filepath.Join(dstDir, "filea.txt"))
		assert.NoFileExists(t, filepath.Join(dstDir, "file10.txt"))
	})

	t.Run("pattern with character class", func(t *testing.T) {
		// Create files for character class testing
		testRepo.CreateFile("test1.log", "log1")
		testRepo.CreateFile("test2.log", "log2")
		testRepo.CreateFile("test9.log", "log9")
		testRepo.CreateFile("testa.log", "loga") // Should not match test[0-9].log
		testRepo.CommitAll("Add character class test files")

		err := os.RemoveAll(dstDir)
		require.NoError(t, err)
		err = os.MkdirAll(dstDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "test[0-9].log",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should match test1.log, test2.log, test9.log but not testa.log
		assert.Len(t, copiedFiles, 3)
		assert.FileExists(t, filepath.Join(dstDir, "test1.log"))
		assert.FileExists(t, filepath.Join(dstDir, "test2.log"))
		assert.FileExists(t, filepath.Join(dstDir, "test9.log"))
		assert.NoFileExists(t, filepath.Join(dstDir, "testa.log"))
	})

	t.Run("empty pattern result", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "**/*.nonexistent",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should return empty result without error
		assert.Empty(t, copiedFiles)
	})

	t.Run("invalid glob pattern", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "[invalid-pattern", // Unclosed bracket
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)

		// Should return error for invalid pattern
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid glob pattern")
		assert.Empty(t, copiedFiles)
	})
}

func TestGlobPatternDetection(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "simple file path",
			path:     "file.txt",
			expected: false,
		},
		{
			name:     "simple directory path",
			path:     "src/components/",
			expected: false,
		},
		{
			name:     "asterisk wildcard",
			path:     "*.json",
			expected: true,
		},
		{
			name:     "question mark wildcard",
			path:     "file?.txt",
			expected: true,
		},
		{
			name:     "character class",
			path:     "test[0-9].log",
			expected: true,
		},
		{
			name:     "double asterisk recursive",
			path:     "**/.cursorrules",
			expected: true,
		},
		{
			name:     "complex pattern",
			path:     "src/**/config/*.json",
			expected: true,
		},
		{
			name:     "escaped special characters",
			path:     "file\\*.txt", // Escaped asterisk (not a glob)
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := AutoCopyItem{Path: tt.path}
			result := item.IsGlobPattern()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobPatternPerformance(t *testing.T) {
	// Create a large directory structure for performance testing
	testRepo := testutil.NewTestGitRepository(t, "perf-test")

	// Create many files and directories
	for i := 0; i < 100; i++ {
		dir := fmt.Sprintf("dir%d", i)
		testRepo.CreateDirectory(dir)

		for j := 0; j < 10; j++ {
			testRepo.CreateFile(filepath.Join(dir, fmt.Sprintf("file%d.json", j)), fmt.Sprintf(`{"id": %d}`, j))
			testRepo.CreateFile(filepath.Join(dir, fmt.Sprintf("file%d.txt", j)), fmt.Sprintf("content %d", j))
		}
	}
	testRepo.CommitAll("Add performance test files")

	dstDir := filepath.Join(testRepo.TempDir, "destination")
	err := os.MkdirAll(dstDir, 0755)
	require.NoError(t, err)

	t.Run("large glob pattern performance", func(t *testing.T) {
		config := &AutoCopyConfig{
			Version: 1,
			Items: []AutoCopyItem{
				{
					Path: "**/*.json",
				},
			},
		}

		copier := NewLegacyAutoCopier()
		copiedFiles, err := copier.CopyFiles(testRepo.RepoDir, dstDir, config)
		require.NoError(t, err)

		// Should find all JSON files (100 dirs * 10 files = 1000 files)
		assert.Len(t, copiedFiles, 1000)

		// Verify a few random files exist
		assert.FileExists(t, filepath.Join(dstDir, "dir0", "file0.json"))
		assert.FileExists(t, filepath.Join(dstDir, "dir50", "file5.json"))
		assert.FileExists(t, filepath.Join(dstDir, "dir99", "file9.json"))
	})
}
