//go:build windows

package editor

import (
	"os/exec"
	"strings"
)

// quitCursor quits Cursor on Windows using taskkill
func (e *CursorEditor) quitCursor() error {
	// Try to quit gracefully first
	cmd := exec.Command("taskkill", "/IM", "Cursor.exe", "/T")
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Force quit if graceful quit fails
	cmd = exec.Command("taskkill", "/F", "/IM", "Cursor.exe", "/T")
	return cmd.Run()
}

// isCursorRunning checks if Cursor is running on Windows
func (e *CursorEditor) isCursorRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Cursor.exe")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "Cursor.exe")
}

// quitVSCode quits VS Code on Windows using taskkill
func (e *VSCodeEditor) quitVSCode() error {
	// Try to quit gracefully first
	cmd := exec.Command("taskkill", "/IM", "Code.exe", "/T")
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Force quit if graceful quit fails
	cmd = exec.Command("taskkill", "/F", "/IM", "Code.exe", "/T")
	return cmd.Run()
}

// isVSCodeRunning checks if VS Code is running on Windows
func (e *VSCodeEditor) isVSCodeRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Code.exe")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "Code.exe")
}

// GetRunningProcesses returns a list of running editor processes on Windows
func GetRunningProcesses() ([]string, error) {
	cmd := exec.Command("tasklist", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "Cursor.exe") || strings.Contains(line, "Code.exe") {
			processes = append(processes, strings.TrimSpace(line))
		}
	}

	return processes, nil
}
