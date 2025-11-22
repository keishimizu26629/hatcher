package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/editor"
	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEditor implements the Editor interface for testing
type MockEditor struct {
	name       string
	command    string
	priority   int
	installed  bool
	running    bool
	openCalled bool
	quitCalled bool
	openError  error
	quitError  error
}

func NewMockEditor(name, command string, priority int, installed bool) *MockEditor {
	return &MockEditor{
		name:      name,
		command:   command,
		priority:  priority,
		installed: installed,
	}
}

func (m *MockEditor) Name() string                { return m.name }
func (m *MockEditor) Command() string             { return m.command }
func (m *MockEditor) Priority() int               { return m.priority }
func (m *MockEditor) IsInstalled() bool           { return m.installed }
func (m *MockEditor) GetVersion() (string, error) { return "1.0.0", nil }
func (m *MockEditor) IsRunning() bool             { return m.running }

func (m *MockEditor) Open(path string) error {
	m.openCalled = true
	return m.openError
}

func (m *MockEditor) OpenInNewWindow(path string) error {
	m.openCalled = true
	return m.openError
}

func (m *MockEditor) Quit() error {
	m.quitCalled = true
	m.running = false
	return m.quitError
}

func (m *MockEditor) SetRunning(running bool) { m.running = running }
func (m *MockEditor) SetOpenError(err error)  { m.openError = err }
func (m *MockEditor) SetQuitError(err error)  { m.quitError = err }

// MockEditorDetector implements editor detection for testing
type MockEditorDetector struct {
	editors    []editor.Editor
	bestEditor editor.Editor
}

func NewMockEditorDetector() *MockEditorDetector {
	return &MockEditorDetector{}
}

func (m *MockEditorDetector) AddEditor(ed editor.Editor) {
	m.editors = append(m.editors, ed)
	if m.bestEditor == nil || ed.Priority() < m.bestEditor.Priority() {
		m.bestEditor = ed
	}
}

func (m *MockEditorDetector) DetectAvailable() []editor.Editor {
	var available []editor.Editor
	for _, ed := range m.editors {
		if ed.IsInstalled() {
			available = append(available, ed)
		}
	}
	return available
}

func (m *MockEditorDetector) GetBestEditor() editor.Editor {
	available := m.DetectAvailable()
	if len(available) > 0 {
		return available[0]
	}
	return nil
}

func (m *MockEditorDetector) GetEditorByName(name string) editor.Editor {
	for _, ed := range m.editors {
		if ed.Command() == name {
			return ed
		}
	}
	return nil
}

func TestMover_MoveToWorktree(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "mover-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create mock editor detector
	mockDetector := NewMockEditorDetector()
	mockEditor := NewMockEditor("Test Editor", "test-editor", 1, true)
	mockDetector.AddEditor(mockEditor)

	// Create mover with mock detector
	mover := NewMover(repo, mockDetector)

	t.Run("move to existing worktree", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/test-move"
		worktreePath := filepath.Join(testRepo.TempDir, "mover-test-feature-test-move")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Test move operation
		options := MoveOptions{
			BranchName: branchName,
			SwitchMode: false,
			AutoCreate: false,
		}

		result, err := mover.MoveToWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, branchName, result.BranchName)
		assert.Equal(t, worktreePath, result.WorktreePath)
		assert.False(t, result.CreatedNew)
		assert.Equal(t, "Test Editor", result.EditorUsed)

		// Verify editor was called
		assert.True(t, mockEditor.openCalled)
		assert.False(t, mockEditor.quitCalled) // No switch mode
	})

	t.Run("move to existing worktree with switch mode", func(t *testing.T) {
		// Reset mock editor
		mockEditor.openCalled = false
		mockEditor.quitCalled = false
		mockEditor.SetRunning(true)

		// Test move operation with switch mode
		options := MoveOptions{
			BranchName: "feature/test-move",
			SwitchMode: true,
			AutoCreate: false,
		}

		result, err := mover.MoveToWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify editor operations
		assert.True(t, mockEditor.quitCalled) // Should quit first
		assert.True(t, mockEditor.openCalled) // Then open
	})

	t.Run("move to non-existent worktree without auto-create", func(t *testing.T) {
		// Test move to non-existent worktree
		options := MoveOptions{
			BranchName: "feature/non-existent",
			SwitchMode: false,
			AutoCreate: false,
		}

		result, err := mover.MoveToWorktree(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "worktree not found")
	})

	t.Run("move to non-existent worktree with auto-create", func(t *testing.T) {
		// Reset mock editor
		mockEditor.openCalled = false
		mockEditor.quitCalled = false

		// Test move with auto-create
		options := MoveOptions{
			BranchName: "feature/auto-create-test",
			SwitchMode: false,
			AutoCreate: true,
		}

		result, err := mover.MoveToWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, "feature/auto-create-test", result.BranchName)
		assert.True(t, result.CreatedNew)
		assert.Equal(t, "Test Editor", result.EditorUsed)

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "mover-test-feature-auto-create-test")
		assert.DirExists(t, expectedPath)

		// Verify editor was called
		assert.True(t, mockEditor.openCalled)
	})

	t.Run("move with specific editor", func(t *testing.T) {
		// Add another mock editor
		specificEditor := NewMockEditor("Specific Editor", "specific-editor", 2, true)
		mockDetector.AddEditor(specificEditor)

		// Reset mock editors
		mockEditor.openCalled = false
		specificEditor.openCalled = false

		// Test move with specific editor
		options := MoveOptions{
			BranchName:    "feature/test-move",
			SwitchMode:    false,
			AutoCreate:    false,
			EditorCommand: "specific-editor",
		}

		result, err := mover.MoveToWorktree(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify specific editor was used
		assert.Equal(t, "Specific Editor", result.EditorUsed)
		assert.True(t, specificEditor.openCalled)
		assert.False(t, mockEditor.openCalled) // Default editor not used
	})

	t.Run("move with no editor available", func(t *testing.T) {
		// Create mover with no editors
		emptyDetector := NewMockEditorDetector()
		emptyMover := NewMover(repo, emptyDetector)

		// Test move operation
		options := MoveOptions{
			BranchName: "feature/test-move",
			SwitchMode: false,
			AutoCreate: false,
		}

		result, err := emptyMover.MoveToWorktree(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no suitable editor found")
	})

	t.Run("move with editor open failure", func(t *testing.T) {
		// Set editor to fail on open
		failingEditor := NewMockEditor("Failing Editor", "failing-editor", 1, true)
		failingEditor.SetOpenError(assert.AnError)

		failingDetector := NewMockEditorDetector()
		failingDetector.AddEditor(failingEditor)
		failingMover := NewMover(repo, failingDetector)

		// Test move operation
		options := MoveOptions{
			BranchName: "feature/test-move",
			SwitchMode: false,
			AutoCreate: false,
		}

		result, err := failingMover.MoveToWorktree(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to open editor")
	})
}

func TestMover_CreateAndMove(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "create-move-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	// Create mock editor detector
	mockDetector := NewMockEditorDetector()
	mockEditor := NewMockEditor("Test Editor", "test-editor", 1, true)
	mockDetector.AddEditor(mockEditor)

	// Create mover
	mover := NewMover(repo, mockDetector)

	t.Run("create and move to new worktree", func(t *testing.T) {
		// Test create and move
		options := CreateAndMoveOptions{
			BranchName: "feature/create-and-move",
			Force:      false,
			NoCopy:     false,
		}

		result, err := mover.CreateAndMove(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, "feature/create-and-move", result.BranchName)
		assert.True(t, result.CreatedNew)
		assert.Equal(t, "Test Editor", result.EditorUsed)

		// Verify worktree was created
		expectedPath := filepath.Join(testRepo.TempDir, "create-move-test-feature-create-and-move")
		assert.DirExists(t, expectedPath)

		// Verify editor was called
		assert.True(t, mockEditor.openCalled)
	})

	t.Run("create and move with existing directory", func(t *testing.T) {
		// Create directory that would conflict
		conflictPath := filepath.Join(testRepo.TempDir, "create-move-test-feature-conflict")
		err := os.MkdirAll(conflictPath, 0755)
		require.NoError(t, err)

		// Test create and move without force
		options := CreateAndMoveOptions{
			BranchName: "feature/conflict",
			Force:      false,
			NoCopy:     false,
		}

		result, err := mover.CreateAndMove(options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "directory already exists")
	})

	t.Run("create and move with force", func(t *testing.T) {
		// Reset mock editor
		mockEditor.openCalled = false

		// Test create and move with force
		options := CreateAndMoveOptions{
			BranchName: "feature/conflict",
			Force:      true,
			NoCopy:     false,
		}

		result, err := mover.CreateAndMove(options)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify result
		assert.Equal(t, "feature/conflict", result.BranchName)
		assert.True(t, result.CreatedNew)

		// Verify editor was called
		assert.True(t, mockEditor.openCalled)
	})
}
