//go:build darwin

package editor

import (
	"os/exec"
	"strings"
)

// quitCursor quits Cursor on macOS using AppleScript
func (e *CursorEditor) quitCursor() error {
	// Try AppleScript first
	script := `tell application "Cursor" to quit`
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Fallback to pkill
	cmd = exec.Command("pkill", "-f", "Cursor")
	return cmd.Run()
}

// isCursorRunning checks if Cursor is running on macOS
func (e *CursorEditor) isCursorRunning() bool {
	// Check using pgrep
	cmd := exec.Command("pgrep", "-f", "Cursor")
	err := cmd.Run()
	return err == nil
}

// quitVSCode quits VS Code on macOS using AppleScript
func (e *VSCodeEditor) quitVSCode() error {
	// Try AppleScript first
	script := `tell application "Visual Studio Code" to quit`
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Fallback to pkill
	cmd = exec.Command("pkill", "-f", "Visual Studio Code")
	return cmd.Run()
}

// isVSCodeRunning checks if VS Code is running on macOS
func (e *VSCodeEditor) isVSCodeRunning() bool {
	// Check using pgrep
	cmd := exec.Command("pgrep", "-f", "Visual Studio Code")
	err := cmd.Run()
	return err == nil
}

// GetRunningProcesses returns a list of running editor processes on macOS
func GetRunningProcesses() ([]string, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "Cursor") || strings.Contains(line, "Visual Studio Code") {
			processes = append(processes, strings.TrimSpace(line))
		}
	}

	return processes, nil
}
