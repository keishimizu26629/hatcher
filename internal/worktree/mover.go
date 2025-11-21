package worktree

import (
	"fmt"
	"time"

	"github.com/keisukeshimizu/hatcher/internal/editor"
	"github.com/keisukeshimizu/hatcher/internal/git"
)

// EditorDetector interface for dependency injection
type EditorDetector interface {
	DetectAvailable() []editor.Editor
	GetBestEditor() editor.Editor
	GetEditorByName(name string) editor.Editor
}

// Mover handles worktree movement and editor integration
type Mover struct {
	repo     git.Repository
	detector EditorDetector
	finder   *Finder
	creator  *Creator
}

// NewMover creates a new worktree mover
func NewMover(repo git.Repository, detector EditorDetector) *Mover {
	return &Mover{
		repo:     repo,
		detector: detector,
		finder:   NewFinder(repo),
		creator:  NewCreator(repo),
	}
}

// MoveOptions contains options for moving to a worktree
type MoveOptions struct {
	BranchName    string
	SwitchMode    bool   // Close current editor and switch
	AutoCreate    bool   // Create worktree if it doesn't exist
	EditorCommand string // Specific editor to use
}

// CreateAndMoveOptions contains options for creating and moving to a worktree
type CreateAndMoveOptions struct {
	BranchName        string
	Force             bool
	NoCopy            bool
	NoGitignoreUpdate bool
	EditorCommand     string
}

// MoveResult contains the result of a move operation
type MoveResult struct {
	BranchName   string    `json:"branchName"`
	WorktreePath string    `json:"worktreePath"`
	CreatedNew   bool      `json:"createdNew"`
	EditorUsed   string    `json:"editorUsed"`
	Timestamp    time.Time `json:"timestamp"`
}

// MoveToWorktree moves to an existing worktree or creates one if requested
func (m *Mover) MoveToWorktree(options MoveOptions) (*MoveResult, error) {
	// Find existing worktree
	worktreePath, exists, err := m.finder.FindWorktree(options.BranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for worktree: %w", err)
	}

	var createdNew bool

	if !exists {
		if !options.AutoCreate {
			return nil, fmt.Errorf("worktree not found for branch '%s' (use --yes to create automatically)", options.BranchName)
		}

		// Create new worktree
		createOptions := CreateOptions{
			BranchName: options.BranchName,
			Force:      false,
			NoCopy:     false, // Enable auto-copy for move operations
			DryRun:     false,
		}

		createResult, err := m.creator.Create(createOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}

		worktreePath = createResult.WorktreePath
		createdNew = true
	}

	// Get editor to use
	selectedEditor, err := m.selectEditor(options.EditorCommand)
	if err != nil {
		return nil, err
	}

	// Handle switch mode (quit current editor first)
	if options.SwitchMode {
		if selectedEditor.IsRunning() {
			if err := selectedEditor.Quit(); err != nil {
				return nil, fmt.Errorf("failed to quit current editor: %w", err)
			}

			// Wait a moment for the editor to fully close
			time.Sleep(1 * time.Second)
		}
	}

	// Open worktree in editor
	if err := selectedEditor.OpenInNewWindow(worktreePath); err != nil {
		return nil, fmt.Errorf("failed to open editor: %w", err)
	}

	return &MoveResult{
		BranchName:   options.BranchName,
		WorktreePath: worktreePath,
		CreatedNew:   createdNew,
		EditorUsed:   selectedEditor.Name(),
		Timestamp:    time.Now(),
	}, nil
}

// CreateAndMove creates a new worktree and opens it in an editor
func (m *Mover) CreateAndMove(options CreateAndMoveOptions) (*MoveResult, error) {
	// Create worktree first
	createOptions := CreateOptions{
		BranchName:        options.BranchName,
		Force:             options.Force,
		NoCopy:            options.NoCopy,
		NoGitignoreUpdate: options.NoGitignoreUpdate,
		DryRun:            false,
	}

	createResult, err := m.creator.Create(createOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Get editor to use
	selectedEditor, err := m.selectEditor(options.EditorCommand)
	if err != nil {
		return nil, err
	}

	// Open worktree in editor
	if err := selectedEditor.OpenInNewWindow(createResult.WorktreePath); err != nil {
		return nil, fmt.Errorf("failed to open editor: %w", err)
	}

	return &MoveResult{
		BranchName:   createResult.BranchName,
		WorktreePath: createResult.WorktreePath,
		CreatedNew:   true,
		EditorUsed:   selectedEditor.Name(),
		Timestamp:    time.Now(),
	}, nil
}

// selectEditor selects the appropriate editor based on options
func (m *Mover) selectEditor(editorCommand string) (editor.Editor, error) {
	if editorCommand != "" {
		// Use specific editor if requested
		selectedEditor := m.detector.GetEditorByName(editorCommand)
		if selectedEditor == nil {
			return nil, fmt.Errorf("editor '%s' not found", editorCommand)
		}
		if !selectedEditor.IsInstalled() {
			return nil, fmt.Errorf("editor '%s' is not installed", editorCommand)
		}
		return selectedEditor, nil
	}

	// Use best available editor
	bestEditor := m.detector.GetBestEditor()
	if bestEditor == nil {
		return nil, fmt.Errorf("no suitable editor found (cursor, code)")
	}

	return bestEditor, nil
}

// GetAvailableEditors returns a list of available editors
func (m *Mover) GetAvailableEditors() []editor.Editor {
	return m.detector.DetectAvailable()
}

// IsEditorRunning checks if any editor is currently running
func (m *Mover) IsEditorRunning() bool {
	editors := m.detector.DetectAvailable()
	for _, ed := range editors {
		if ed.IsRunning() {
			return true
		}
	}
	return false
}

// GetRunningEditor returns the currently running editor, if any
func (m *Mover) GetRunningEditor() editor.Editor {
	editors := m.detector.DetectAvailable()
	for _, ed := range editors {
		if ed.IsRunning() {
			return ed
		}
	}
	return nil
}
