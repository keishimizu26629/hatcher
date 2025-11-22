package autocopy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/require"
)

// BenchmarkParallelCopy benchmarks parallel copying performance
func BenchmarkParallelCopy(b *testing.B) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(b, "perf-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(b, err)

	// Create test files
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("file%d.txt", i))
		content := strings.Repeat(fmt.Sprintf("content %d ", i), 100)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(b, err)
	}

	config := &AutoCopyConfig{
		Version: 2,
		Items: []AutoCopyItem{
			{Path: "file*.txt", Directory: boolPtr(false), UseGlob: true},
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		destDir := filepath.Join(testRepo.TempDir, fmt.Sprintf("bench-dest-%d", i))
		err := os.MkdirAll(destDir, 0755)
		require.NoError(b, err)

		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers: 4,
		})

		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(b, err)
	}
}

// BenchmarkSequentialVsParallel compares sequential vs parallel performance
func BenchmarkSequentialVsParallel(b *testing.B) {
	testCases := []struct {
		name      string
		workers   int
		fileCount int
		fileSize  int
	}{
		{"Sequential_10files", 1, 10, 1024},
		{"Parallel4_10files", 4, 10, 1024},
		{"Sequential_100files", 1, 100, 1024},
		{"Parallel4_100files", 4, 100, 1024},
		{"Sequential_1000files", 1, 1000, 512},
		{"Parallel4_1000files", 4, 1000, 512},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test repository
			testRepo := helpers.NewTestGitRepository(b, "perf-compare")
			repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
			require.NoError(b, err)

			// Create test files
			for i := 0; i < tc.fileCount; i++ {
				filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("file%d.txt", i))
				content := strings.Repeat("x", tc.fileSize)
				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(b, err)
			}

			config := &AutoCopyConfig{
				Version: 2,
				Items: []AutoCopyItem{
					{Path: "file*.txt", Directory: boolPtr(false), UseGlob: true},
				},
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				destDir := filepath.Join(testRepo.TempDir, fmt.Sprintf("bench-dest-%d", i))
				err := os.MkdirAll(destDir, 0755)
				require.NoError(b, err)

				copier := NewParallelCopier(repo, config, ParallelCopyOptions{
					MaxWorkers: tc.workers,
				})

				err = copier.Run(testRepo.RepoDir, destDir)
				require.NoError(b, err)
			}
		})
	}
}

// BenchmarkMemoryUsage tests memory usage during large file operations
func BenchmarkMemoryUsage(b *testing.B) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(b, "memory-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(b, err)

	// Create large files
	numFiles := 50
	fileSize := 1024 * 1024 // 1MB each
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("large%d.bin", i))
		content := make([]byte, fileSize)
		for j := range content {
			content[j] = byte(j % 256)
		}
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(b, err)
	}

	config := &AutoCopyConfig{
		Version: 2,
		Items: []AutoCopyItem{
			{Path: "large*.bin", Directory: boolPtr(false), UseGlob: true},
		},
	}

	// Measure memory before
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		destDir := filepath.Join(testRepo.TempDir, fmt.Sprintf("memory-dest-%d", i))
		err := os.MkdirAll(destDir, 0755)
		require.NoError(b, err)

		copier := NewParallelCopier(repo, config, ParallelCopyOptions{
			MaxWorkers: 2,
			BufferSize: 64 * 1024, // 64KB buffer
		})

		err = copier.Run(testRepo.RepoDir, destDir)
		require.NoError(b, err)
	}

	b.StopTimer()

	// Measure memory after
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "regression-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create test files
	numFiles := 500
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("regress%d.txt", i))
		content := strings.Repeat(fmt.Sprintf("regression test %d ", i), 50)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	config := &AutoCopyConfig{
		Version: 2,
		Items: []AutoCopyItem{
			{Path: "regress*.txt", Directory: boolPtr(false), UseGlob: true},
		},
	}

	destDir := filepath.Join(testRepo.TempDir, "regression-dest")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	// Test with different worker counts
	workerCounts := []int{1, 2, 4, 8}
	results := make(map[int]time.Duration)

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
		results[workers] = duration

		t.Logf("Workers: %d, Duration: %v", workers, duration)
	}

	// Check that parallel processing is faster than sequential
	sequential := results[1]
	parallel4 := results[4]

	if parallel4 >= sequential {
		t.Logf("Warning: Parallel processing (4 workers: %v) not faster than sequential (%v)", parallel4, sequential)
		// Don't fail the test, but log the warning
	} else {
		speedup := float64(sequential) / float64(parallel4)
		t.Logf("Speedup with 4 workers: %.2fx", speedup)
	}

	// Performance threshold: should complete within reasonable time
	maxDuration := 30 * time.Second
	if results[4] > maxDuration {
		t.Errorf("Performance regression: copying %d files took %v, expected < %v", numFiles, results[4], maxDuration)
	}
}

// TestConcurrentSafety tests concurrent access safety
func TestConcurrentSafety(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "concurrent-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create test files
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(testRepo.RepoDir, fmt.Sprintf("concurrent%d.txt", i))
		content := fmt.Sprintf("concurrent test file %d", i)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	config := &AutoCopyConfig{
		Version: 2,
		Items: []AutoCopyItem{
			{Path: "concurrent*.txt", Directory: boolPtr(false), UseGlob: true},
		},
	}

	// Run multiple concurrent copy operations
	numConcurrent := 5
	done := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			destDir := filepath.Join(testRepo.TempDir, fmt.Sprintf("concurrent-dest-%d", id))
			err := os.MkdirAll(destDir, 0755)
			if err != nil {
				done <- err
				return
			}

			copier := NewParallelCopier(repo, config, ParallelCopyOptions{
				MaxWorkers: 2,
			})

			err = copier.Run(testRepo.RepoDir, destDir)
			done <- err
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numConcurrent; i++ {
		err := <-done
		require.NoError(t, err, "Concurrent operation %d failed", i)
	}

	// Verify all files were copied correctly in each destination
	for i := 0; i < numConcurrent; i++ {
		destDir := filepath.Join(testRepo.TempDir, fmt.Sprintf("concurrent-dest-%d", i))
		files, err := filepath.Glob(filepath.Join(destDir, "concurrent*.txt"))
		require.NoError(t, err)
		require.Equal(t, numFiles, len(files), "Incorrect number of files in destination %d", i)
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
