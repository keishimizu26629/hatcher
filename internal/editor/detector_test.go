package editor

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorDetector_DetectAvailable(t *testing.T) {
	detector := NewDetector()

	t.Run("detect available editors", func(t *testing.T) {
		// This test will vary based on what's actually installed
		editors := detector.DetectAvailable()

		// Should return a list (may be empty if no editors installed)
		assert.NotNil(t, editors)

		// If any editors are found, they should have valid properties
		for _, editor := range editors {
			assert.NotEmpty(t, editor.Name())
			assert.NotEmpty(t, editor.Command())
			assert.True(t, editor.IsInstalled())
			assert.GreaterOrEqual(t, editor.Priority(), 1)
		}
	})

	t.Run("editors sorted by priority", func(t *testing.T) {
		editors := detector.DetectAvailable()

		// Verify editors are sorted by priority (lower number = higher priority)
		for i := 1; i < len(editors); i++ {
			assert.LessOrEqual(t, editors[i-1].Priority(), editors[i].Priority())
		}
	})

	t.Run("detect specific editor", func(t *testing.T) {
		// Test with a command that should exist on most systems
		testCommand := "echo" // This should exist on Unix systems

		if isCommandAvailable(testCommand) {
			// Create a mock editor info for testing
			mockEditor := &EditorInfo{
				Name:        "Test Editor",
				Command:     testCommand,
				VersionFlag: "--version",
				Priority:    99,
			}

			editor := NewEditor(mockEditor)
			assert.True(t, editor.IsInstalled())
			assert.Equal(t, "Test Editor", editor.Name())
			assert.Equal(t, testCommand, editor.Command())
		}
	})
}

func TestEditorDetector_GetBestEditor(t *testing.T) {
	detector := NewDetector()

	t.Run("get best available editor", func(t *testing.T) {
		editor := detector.GetBestEditor()

		if editor != nil {
			// If an editor is found, it should be valid
			assert.NotEmpty(t, editor.Name())
			assert.NotEmpty(t, editor.Command())
			assert.True(t, editor.IsInstalled())
		}
		// If no editor found, that's also valid (returns nil)
	})

	t.Run("get editor by name", func(t *testing.T) {
		// Test getting editor by name
		editor := detector.GetEditorByName("cursor")

		if editor != nil {
			assert.Equal(t, "Cursor", editor.Name())
			assert.Equal(t, "cursor", editor.Command())
		}

		// Test with non-existent editor
		nonExistent := detector.GetEditorByName("non-existent-editor")
		assert.Nil(t, nonExistent)
	})
}

func TestEditorInfo_IsInstalled(t *testing.T) {
	t.Run("installed command", func(t *testing.T) {
		// Use a command that should exist
		info := &EditorInfo{
			Name:        "Test",
			Command:     "echo",
			VersionFlag: "--version",
			Priority:    1,
		}

		editor := NewEditor(info)
		if isCommandAvailable("echo") {
			assert.True(t, editor.IsInstalled())
		}
	})

	t.Run("non-existent command", func(t *testing.T) {
		info := &EditorInfo{
			Name:        "Non-existent",
			Command:     "non-existent-command-12345",
			VersionFlag: "--version",
			Priority:    1,
		}

		editor := NewEditor(info)
		assert.False(t, editor.IsInstalled())
	})
}

func TestEditorInfo_GetVersion(t *testing.T) {
	t.Run("get version from installed command", func(t *testing.T) {
		// Skip if not on a Unix-like system
		if !isCommandAvailable("echo") {
			t.Skip("echo command not available")
		}

		info := &EditorInfo{
			Name:        "Test",
			Command:     "echo",
			VersionFlag: "test-version",
			Priority:    1,
		}

		editor := NewEditor(info)
		version, err := editor.GetVersion()

		// Should succeed and return the echo output
		require.NoError(t, err)
		assert.Contains(t, version, "test-version")
	})

	t.Run("get version from non-existent command", func(t *testing.T) {
		info := &EditorInfo{
			Name:        "Non-existent",
			Command:     "non-existent-command-12345",
			VersionFlag: "--version",
			Priority:    1,
		}

		editor := NewEditor(info)
		version, err := editor.GetVersion()

		// Should fail
		assert.Error(t, err)
		assert.Empty(t, version)
	})
}

func TestCursorEditor(t *testing.T) {
	t.Run("cursor editor properties", func(t *testing.T) {
		detector := NewDetector()
		editor := detector.GetEditorByName("cursor")

		if editor != nil {
			assert.Equal(t, "Cursor", editor.Name())
			assert.Equal(t, "cursor", editor.Command())
			assert.Equal(t, 1, editor.Priority()) // Highest priority
		}
	})

	t.Run("cursor editor operations", func(t *testing.T) {
		// Skip if Cursor is not installed
		if !isCommandAvailable("cursor") {
			t.Skip("Cursor not installed")
		}

		detector := NewDetector()
		editor := detector.GetEditorByName("cursor")
		require.NotNil(t, editor)

		// Test version retrieval
		version, err := editor.GetVersion()
		if err == nil {
			assert.NotEmpty(t, version)
		}

		// Test installation check
		assert.True(t, editor.IsInstalled())
	})
}

func TestVSCodeEditor(t *testing.T) {
	t.Run("vscode editor properties", func(t *testing.T) {
		detector := NewDetector()
		editor := detector.GetEditorByName("code")

		if editor != nil {
			assert.Equal(t, "VS Code", editor.Name())
			assert.Equal(t, "code", editor.Command())
			assert.Equal(t, 2, editor.Priority()) // Second priority
		}
	})

	t.Run("vscode editor operations", func(t *testing.T) {
		// Skip if VS Code is not installed
		if !isCommandAvailable("code") {
			t.Skip("VS Code not installed")
		}

		detector := NewDetector()
		editor := detector.GetEditorByName("code")
		require.NotNil(t, editor)

		// Test version retrieval
		version, err := editor.GetVersion()
		if err == nil {
			assert.NotEmpty(t, version)
		}

		// Test installation check
		assert.True(t, editor.IsInstalled())
	})
}

func TestEditorPriority(t *testing.T) {
	detector := NewDetector()

	t.Run("cursor has higher priority than vscode", func(t *testing.T) {
		cursor := detector.GetEditorByName("cursor")
		vscode := detector.GetEditorByName("code")

		if cursor != nil && vscode != nil {
			assert.Less(t, cursor.Priority(), vscode.Priority())
		}
	})
}

func TestEditorDetection_Integration(t *testing.T) {
	t.Run("full detection workflow", func(t *testing.T) {
		detector := NewDetector()

		// Get all available editors
		editors := detector.DetectAvailable()

		// Get the best editor
		bestEditor := detector.GetBestEditor()

		if len(editors) > 0 {
			// Best editor should be the first in the list (highest priority)
			assert.Equal(t, editors[0], bestEditor)

			// All editors should be installed
			for _, editor := range editors {
				assert.True(t, editor.IsInstalled())
			}
		} else {
			// No editors found
			assert.Nil(t, bestEditor)
		}
	})

	t.Run("editor command execution safety", func(t *testing.T) {
		// Test that we don't accidentally execute dangerous commands
		detector := NewDetector()

		// This should not exist and should be safely handled
		editor := detector.GetEditorByName("rm") // Dangerous command
		assert.Nil(t, editor) // Should not be in our predefined list
	})
}

// Helper function to check if a command is available
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
