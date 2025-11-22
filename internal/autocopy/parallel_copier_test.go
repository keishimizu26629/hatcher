package autocopy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelCopier_Run(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "parallel-copier-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("parallel copy multiple files", func(t *testing.T) {
		// Create source files
		sourceFiles := []string{
			".ai/prompts.md",
			".ai/context.md",
			".cursorrules",
			".clinerules",
			"CLAUDE.md",
		}

		for _, file := range sourceFiles {
			filePath := filepath.Join(testRepo.RepoDir, file)
			if filepath.Dir(file) != "." {
				err := os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)
			}
			err := os.WriteFile(filePath, []byte("test content for "+file), 0644)
			require.NoError(t, err)
		}

		// Create destination directory
		destDir := filepath.Join(testRepo.TempDir, "parallel-dest")
		err := os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		// Create config
		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: ".ai/", Directory: testutil.BoolPtr(true), Recursive: true},
				{Path: ".cursorrules", Directory: testutil.BoolPtr(false)},
				{Path: ".clinerules", Directory: testutil.BoolPtr(false)},
				{Path: "CLAUDE.md", Directory: testutil.BoolPtr(false)},
			},
		}

		// Create parallel copier
		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers:      4,
			BufferSize:      1024,
			ShowProgress:    false,
			VerifyIntegrity: true,
		})

		// Measure execution time
		start := time.Now()
		err = copier.Run(testRepo.RepoDir, destDir)
		duration := time.Since(start)

		require.NoError(t, err)
		t.Logf("Parallel copy took: %v", duration)

		// Verify all files were copied
		for _, file := range sourceFiles {
			destPath := filepath.Join(destDir, file)
			assert.FileExists(t, destPath)

			// Verify content
			content, err := os.ReadFile(destPath)
			require.NoError(t, err)
			assert.Equal(t, "test content for "+file, string(content))
		}
	})

	t.Run("parallel copy with progress tracking", func(t *testing.T) {
		// Create many small files
		numFiles := 20
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("file%d.txt", i))
			err := os.WriteFile(filePath, []byte(fmt.Sprintf("content %d", i)), 0644)
			require.NoError(t, err)
		}

		destDir := filepath.Join(testRepo.TempDir, "progress-dest")
		err := os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		// Create config with glob pattern
		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: "file*.txt", Directory: testutil.BoolPtr(false), UseGlob: true},
			},
		}

		// Track progress
		var progressUpdates []ProgressUpdate
		var progressMutex sync.Mutex

		progressCallback := func(update ProgressUpdate) {
			progressMutex.Lock()
			progressUpdates = append(progressUpdates, update)
			progressMutex.Unlock()
		}

		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers:       2,
			ShowProgress:     true,
			ProgressCallback: progressCallback,
		})

		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(t, err)

		// Verify progress updates
		progressMutex.Lock()
		assert.NotEmpty(t, progressUpdates)

		// Should have start and completion updates
		assert.Equal(t, ProgressTypeStart, progressUpdates[0].Type)
		assert.Equal(t, ProgressTypeComplete, progressUpdates[len(progressUpdates)-1].Type)
		progressMutex.Unlock()

		// Verify all files were copied
		for i := 0; i < numFiles; i++ {
			destPath := filepath.Join(destDir, fmt.Sprintf("file%d.txt", i))
			assert.FileExists(t, destPath)
		}
	})

	t.Run("parallel copy with integrity verification", func(t *testing.T) {
		// Create files with different sizes
		testFiles := map[string]string{
			"small.txt":  "small content",
			"medium.txt": strings.Repeat("medium content ", 100),
			"large.txt":  strings.Repeat("large content ", 1000),
		}

		for filename, content := range testFiles {
			filePath := filepath.Join(testRepo.RepoDir, filename)
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		destDir := filepath.Join(testRepo.TempDir, "integrity-dest")
		err := os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: "*.txt", Directory: testutil.BoolPtr(false), UseGlob: true},
			},
		}

		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers:      2,
			VerifyIntegrity: true,
			ChecksumType:    "sha256",
		})

		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(t, err)

		// Verify all files were copied with correct content
		for filename, expectedContent := range testFiles {
			destPath := filepath.Join(destDir, filename)
			content, err := os.ReadFile(destPath)
			require.NoError(t, err)
			assert.Equal(t, expectedContent, string(content))
		}
	})

	t.Run("parallel copy with error handling", func(t *testing.T) {
		// Create some valid files and some problematic ones
		validFile := filepath.Join(testRepo.RepoDir, "valid.txt")
		err := os.WriteFile(validFile, []byte("valid content"), 0644)
		require.NoError(t, err)

		destDir := filepath.Join(testRepo.TempDir, "error-dest")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: "valid.txt", Directory: testutil.BoolPtr(false)},
				{Path: "nonexistent.txt", Directory: testutil.BoolPtr(false)}, // This will fail
			},
		}

		var errorCount int
		var errorMutex sync.Mutex

		errorCallback := func(err CopyError) {
			errorMutex.Lock()
			errorCount++
			errorMutex.Unlock()
		}

		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers:      2,
			ContinueOnError: true,
			ErrorCallback:   errorCallback,
		})

		err = copier.Run(testRepo.RepoDir, destDir)
		// Should not fail completely due to ContinueOnError
		require.NoError(t, err)

		// Verify valid file was copied
		assert.FileExists(t, filepath.Join(destDir, "valid.txt"))

		// Verify error was reported
		errorMutex.Lock()
		assert.Greater(t, errorCount, 0)
		errorMutex.Unlock()
	})

	t.Run("parallel copy performance comparison", func(t *testing.T) {
		// Create many files for performance testing
		numFiles := 50
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("perf%d.txt", i))
			content := strings.Repeat(fmt.Sprintf("performance test content %d ", i), 10)
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: "perf*.txt", Directory: testutil.BoolPtr(false), UseGlob: true},
			},
		}

		// Test sequential copy
		seqDestDir := filepath.Join(testRepo.TempDir, "seq-dest")
		err := os.MkdirAll(seqDestDir, 0755)
		require.NoError(t, err)

		seqCopier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers: 1, // Sequential
		})

		seqStart := time.Now()
		err = seqCopier.Run(testRepo.RepoDir, seqDestDir)
		seqDuration := time.Since(seqStart)
		require.NoError(t, err)

		// Test parallel copy
		parDestDir := filepath.Join(testRepo.TempDir, "par-dest")
		err = os.MkdirAll(parDestDir, 0755)
		require.NoError(t, err)

		parCopier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers: 4, // Parallel
		})

		parStart := time.Now()
		err = parCopier.Run(testRepo.RepoDir, parDestDir)
		parDuration := time.Since(parStart)
		require.NoError(t, err)

		t.Logf("Sequential copy: %v", seqDuration)
		t.Logf("Parallel copy: %v", parDuration)
		t.Logf("Speedup: %.2fx", float64(seqDuration)/float64(parDuration))

		// Verify both copied the same number of files
		seqFiles, err := filepath.Glob(filepath.Join(seqDestDir, "perf*.txt"))
		require.NoError(t, err)
		parFiles, err := filepath.Glob(filepath.Join(parDestDir, "perf*.txt"))
		require.NoError(t, err)
		assert.Equal(t, len(seqFiles), len(parFiles))
		assert.Equal(t, numFiles, len(parFiles))
	})
}

func TestParallelCopier_WorkerPool(t *testing.T) {
	testRepo := testutil.NewTestGitRepository(t, "worker-pool-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("worker pool scaling", func(t *testing.T) {
		// Create files
		numFiles := 10
		for i := 0; i < numFiles; i++ {
			filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("worker%d.txt", i))
			err := os.WriteFile(filePath, []byte(fmt.Sprintf("worker test %d", i)), 0644)
			require.NoError(t, err)
		}

		destDir := filepath.Join(testRepo.TempDir, "worker-dest")
		err := os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		config := &AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{Path: "worker*.txt", Directory: testutil.BoolPtr(false), UseGlob: true},
			},
		}

		// Test different worker counts
		workerCounts := []int{1, 2, 4, 8}
		for _, workers := range workerCounts {
			subDestDir := filepath.Join(destDir, fmt.Sprintf("workers-%d", workers))
			err := os.MkdirAll(subDestDir, 0755)
			require.NoError(t, err)

			copier := NewParallelCopier(repo, config, ParallelCopyOptions{
				MaxWorkers: workers,
			})

			start := time.Now()
			err = copier.Run(testRepo.RepoDir, subDestDir)
			duration := time.Since(start)

			require.NoError(t, err)
			t.Logf("Workers: %d, Duration: %v", workers, duration)

			// Verify all files copied
			files, err := filepath.Glob(filepath.Join(subDestDir, "worker*.txt"))
			require.NoError(t, err)
			assert.Equal(t, numFiles, len(files))
		}
	})
}

// Helper function
