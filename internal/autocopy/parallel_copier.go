package autocopy

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// ProgressType represents the type of progress update
type ProgressType string

const (
	ProgressTypeStart    ProgressType = "start"
	ProgressTypeProgress ProgressType = "progress"
	ProgressTypeComplete ProgressType = "complete"
	ProgressTypeError    ProgressType = "error"
)

// ProgressUpdate represents a progress update during copying
type ProgressUpdate struct {
	Type         ProgressType  `json:"type"`
	Message      string        `json:"message"`
	Current      int           `json:"current"`
	Total        int           `json:"total"`
	Percentage   float64       `json:"percentage"`
	BytesCopied  int64         `json:"bytesCopied"`
	TotalBytes   int64         `json:"totalBytes"`
	ElapsedTime  time.Duration `json:"elapsedTime"`
	EstimatedETA time.Duration `json:"estimatedETA"`
}

// CopyError represents an error during copying
type CopyError struct {
	SourcePath string    `json:"sourcePath"`
	DestPath   string    `json:"destPath"`
	Error      error     `json:"error"`
	Timestamp  time.Time `json:"timestamp"`
}

// CopyTask represents a single copy operation
type CopyTask struct {
	SourcePath string
	DestPath   string
	IsDir      bool
	Size       int64
}

// ParallelCopyOptions contains options for parallel copying
type ParallelCopyOptions struct {
	MaxWorkers       int                  // Maximum number of worker goroutines
	BufferSize       int                  // Buffer size for file copying
	ShowProgress     bool                 // Whether to show progress updates
	VerifyIntegrity  bool                 // Whether to verify file integrity after copying
	ChecksumType     string               // Type of checksum to use (sha256, md5)
	ContinueOnError  bool                 // Whether to continue on individual file errors
	ProgressCallback func(ProgressUpdate) // Callback for progress updates
	ErrorCallback    func(CopyError)      // Callback for errors
}

// ParallelCopier handles parallel file copying operations
type ParallelCopier struct {
	repo    git.Repository
	config  *AutoCopyConfig
	options ParallelCopyOptions

	// Internal state
	taskQueue      chan CopyTask
	results        chan error
	progress       chan ProgressUpdate
	errors         chan CopyError
	wg             sync.WaitGroup
	totalTasks     int
	completedTasks int
	totalBytes     int64
	copiedBytes    int64
	startTime      time.Time
	mutex          sync.RWMutex
}

// NewParallelCopier creates a new parallel copier
func NewParallelCopier(repo git.Repository, config *AutoCopyConfig, options ParallelCopyOptions) *ParallelCopier {
	// Set default options
	if options.MaxWorkers <= 0 {
		options.MaxWorkers = 4
	}
	if options.BufferSize <= 0 {
		options.BufferSize = 64 * 1024 // 64KB
	}
	if options.ChecksumType == "" {
		options.ChecksumType = "sha256"
	}

	return &ParallelCopier{
		repo:    repo,
		config:  config,
		options: options,
	}
}

// Run executes the parallel copy operation
func (pc *ParallelCopier) Run(sourceDir, destDir string) error {
	pc.startTime = time.Now()

	// Initialize channels
	pc.taskQueue = make(chan CopyTask, pc.options.MaxWorkers*2)
	pc.results = make(chan error, pc.options.MaxWorkers)
	pc.progress = make(chan ProgressUpdate, 100)
	pc.errors = make(chan CopyError, 100)

	// Start progress handler if needed
	var progressWg sync.WaitGroup
	if pc.options.ShowProgress && pc.options.ProgressCallback != nil {
		progressWg.Add(1)
		go pc.handleProgress(&progressWg)
	}

	// Start error handler if needed
	var errorWg sync.WaitGroup
	if pc.options.ErrorCallback != nil {
		errorWg.Add(1)
		go pc.handleErrors(&errorWg)
	}

	// Discover all copy tasks
	tasks, err := pc.discoverTasks(sourceDir, destDir)
	if err != nil {
		return fmt.Errorf("failed to discover copy tasks: %w", err)
	}

	pc.totalTasks = len(tasks)
	if pc.totalTasks == 0 {
		return nil // Nothing to copy
	}

	// Calculate total bytes
	for _, task := range tasks {
		pc.totalBytes += task.Size
	}

	// Send start progress update
	if pc.options.ShowProgress {
		pc.sendProgressUpdate(ProgressUpdate{
			Type:    ProgressTypeStart,
			Message: fmt.Sprintf("Starting parallel copy of %d items", pc.totalTasks),
			Total:   pc.totalTasks,
		})
	}

	// Start workers
	for i := 0; i < pc.options.MaxWorkers; i++ {
		pc.wg.Add(1)
		go pc.worker()
	}

	// Send tasks to workers
	go func() {
		defer close(pc.taskQueue)
		for _, task := range tasks {
			pc.taskQueue <- task
		}
	}()

	// Wait for all workers to complete
	pc.wg.Wait()

	// Send completion progress update before closing channels
	if pc.options.ShowProgress {
		pc.sendProgressUpdate(ProgressUpdate{
			Type:        ProgressTypeComplete,
			Message:     "Copy operation completed",
			Current:     pc.completedTasks,
			Total:       pc.totalTasks,
			Percentage:  100.0,
			BytesCopied: pc.copiedBytes,
			TotalBytes:  pc.totalBytes,
			ElapsedTime: time.Since(pc.startTime),
		})
	}

	// Close channels
	close(pc.progress)
	close(pc.errors)

	// Wait for handlers to finish
	progressWg.Wait()
	errorWg.Wait()

	return nil
}

// discoverTasks discovers all copy tasks based on the configuration
func (pc *ParallelCopier) discoverTasks(sourceDir, destDir string) ([]CopyTask, error) {
	var tasks []CopyTask

	for _, item := range pc.config.Items {
		itemTasks, err := pc.discoverItemTasks(sourceDir, destDir, item)
		if err != nil {
			if pc.options.ContinueOnError {
				pc.sendError(CopyError{
					SourcePath: item.Path,
					Error:      err,
					Timestamp:  time.Now(),
				})
				continue
			}
			return nil, err
		}
		tasks = append(tasks, itemTasks...)
	}

	return tasks, nil
}

// discoverItemTasks discovers copy tasks for a single configuration item
func (pc *ParallelCopier) discoverItemTasks(sourceDir, destDir string, item AutoCopyItem) ([]CopyTask, error) {
	var tasks []CopyTask

	sourcePath := filepath.Join(sourceDir, item.Path)

	// Handle glob patterns
	if item.UseGlob {
		matches, err := filepath.Glob(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("glob pattern failed for %s: %w", item.Path, err)
		}

		for _, match := range matches {
			relPath, err := filepath.Rel(sourceDir, match)
			if err != nil {
				continue
			}

			itemTasks, err := pc.discoverSinglePath(sourceDir, destDir, relPath, item)
			if err != nil {
				if pc.options.ContinueOnError {
					continue
				}
				return nil, err
			}
			tasks = append(tasks, itemTasks...)
		}
	} else {
		itemTasks, err := pc.discoverSinglePath(sourceDir, destDir, item.Path, item)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, itemTasks...)
	}

	return tasks, nil
}

// discoverSinglePath discovers copy tasks for a single path
func (pc *ParallelCopier) discoverSinglePath(sourceDir, destDir, relativePath string, item AutoCopyItem) ([]CopyTask, error) {
	var tasks []CopyTask

	sourcePath := filepath.Join(sourceDir, relativePath)
	destPath := filepath.Join(destDir, relativePath)

	// Check if source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) && item.AutoDetect {
			return tasks, nil // Skip non-existent files when auto-detecting
		}
		return nil, fmt.Errorf("failed to stat %s: %w", sourcePath, err)
	}

	if info.IsDir() {
		// Handle directory
		if item.Directory != nil && !*item.Directory {
			return nil, fmt.Errorf("expected file but found directory: %s", sourcePath)
		}

		// Add directory creation task
		tasks = append(tasks, CopyTask{
			SourcePath: sourcePath,
			DestPath:   destPath,
			IsDir:      true,
			Size:       0,
		})

		// Recursively add files if needed
		if item.Recursive {
			err := filepath.Walk(sourcePath, func(walkPath string, walkInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				if walkPath == sourcePath {
					return nil // Skip the root directory itself
				}

				relWalkPath, err := filepath.Rel(sourcePath, walkPath)
				if err != nil {
					return err
				}

				destWalkPath := filepath.Join(destPath, relWalkPath)

				if walkInfo.IsDir() {
					tasks = append(tasks, CopyTask{
						SourcePath: walkPath,
						DestPath:   destWalkPath,
						IsDir:      true,
						Size:       0,
					})
				} else {
					tasks = append(tasks, CopyTask{
						SourcePath: walkPath,
						DestPath:   destWalkPath,
						IsDir:      false,
						Size:       walkInfo.Size(),
					})
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", sourcePath, err)
			}
		}
	} else {
		// Handle file
		if item.Directory != nil && *item.Directory {
			return nil, fmt.Errorf("expected directory but found file: %s", sourcePath)
		}

		tasks = append(tasks, CopyTask{
			SourcePath: sourcePath,
			DestPath:   destPath,
			IsDir:      false,
			Size:       info.Size(),
		})
	}

	return tasks, nil
}

// worker is a worker goroutine that processes copy tasks
func (pc *ParallelCopier) worker() {
	defer pc.wg.Done()

	for task := range pc.taskQueue {
		err := pc.processTask(task)
		if err != nil {
			pc.sendError(CopyError{
				SourcePath: task.SourcePath,
				DestPath:   task.DestPath,
				Error:      err,
				Timestamp:  time.Now(),
			})

			if !pc.options.ContinueOnError {
				pc.results <- err
				return
			}
		}

		// Update progress
		pc.mutex.Lock()
		pc.completedTasks++
		pc.copiedBytes += task.Size
		current := pc.completedTasks
		total := pc.totalTasks
		copied := pc.copiedBytes
		totalBytes := pc.totalBytes
		pc.mutex.Unlock()

		// Send progress update
		if pc.options.ShowProgress && current%10 == 0 { // Update every 10 files
			elapsed := time.Since(pc.startTime)
			percentage := float64(current) / float64(total) * 100

			var eta time.Duration
			if current > 0 {
				eta = time.Duration(float64(elapsed) * (float64(total) - float64(current)) / float64(current))
			}

			pc.sendProgressUpdate(ProgressUpdate{
				Type:         ProgressTypeProgress,
				Message:      fmt.Sprintf("Copied %d/%d files", current, total),
				Current:      current,
				Total:        total,
				Percentage:   percentage,
				BytesCopied:  copied,
				TotalBytes:   totalBytes,
				ElapsedTime:  elapsed,
				EstimatedETA: eta,
			})
		}
	}
}

// processTask processes a single copy task
func (pc *ParallelCopier) processTask(task CopyTask) error {
	if task.IsDir {
		// Create directory
		return os.MkdirAll(task.DestPath, 0755)
	}

	// Copy file
	return pc.copyFile(task.SourcePath, task.DestPath)
}

// copyFile copies a single file with optional integrity verification
func (pc *ParallelCopier) copyFile(sourcePath, destPath string) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy with optional integrity verification
	if pc.options.VerifyIntegrity {
		return pc.copyWithVerification(sourceFile, destFile, sourcePath, destPath)
	}

	// Simple copy
	_, err = io.CopyBuffer(destFile, sourceFile, make([]byte, pc.options.BufferSize))
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// copyWithVerification copies a file and verifies its integrity
func (pc *ParallelCopier) copyWithVerification(sourceFile, destFile *os.File, sourcePath, destPath string) error {
	var sourceHash, destHash hash.Hash

	switch pc.options.ChecksumType {
	case "sha256":
		sourceHash = sha256.New()
		destHash = sha256.New()
	default:
		return fmt.Errorf("unsupported checksum type: %s", pc.options.ChecksumType)
	}

	// Create multi-writers for hashing during copy
	sourceReader := io.TeeReader(sourceFile, sourceHash)
	destWriter := io.MultiWriter(destFile, destHash)

	// Copy with hashing
	_, err := io.CopyBuffer(destWriter, sourceReader, make([]byte, pc.options.BufferSize))
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Compare checksums
	sourceChecksum := sourceHash.Sum(nil)
	destChecksum := destHash.Sum(nil)

	if !equalBytes(sourceChecksum, destChecksum) {
		return fmt.Errorf("integrity verification failed: checksums don't match")
	}

	return nil
}

// sendProgressUpdate sends a progress update
func (pc *ParallelCopier) sendProgressUpdate(update ProgressUpdate) {
	if pc.options.ShowProgress && pc.progress != nil {
		select {
		case pc.progress <- update:
		default:
			// Don't block if channel is full
		}
	}
}

// sendError sends an error
func (pc *ParallelCopier) sendError(err CopyError) {
	select {
	case pc.errors <- err:
	default:
		// Don't block if channel is full
	}
}

// handleProgress handles progress updates
func (pc *ParallelCopier) handleProgress(wg *sync.WaitGroup) {
	defer wg.Done()

	for update := range pc.progress {
		if pc.options.ProgressCallback != nil {
			pc.options.ProgressCallback(update)
		}
	}
}

// handleErrors handles errors
func (pc *ParallelCopier) handleErrors(wg *sync.WaitGroup) {
	defer wg.Done()

	for err := range pc.errors {
		if pc.options.ErrorCallback != nil {
			pc.options.ErrorCallback(err)
		}
	}
}

// equalBytes compares two byte slices for equality
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
