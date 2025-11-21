//go:build linux

package editor

import (
	"os/exec"
	"strings"
)

// quitCursor quits Cursor on Linux using pkill
func (e *CursorEditor) quitCursor() error {
	// Try SIGTERM first (graceful)
	cmd := exec.Command("pkill", "-TERM", "-f", "cursor")
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Fallback to SIGKILL (force)
	cmd = exec.Command("pkill", "-KILL", "-f", "cursor")
	return cmd.Run()
}

// isCursorRunning checks if Cursor is running on Linux
func (e *CursorEditor) isCursorRunning() bool {
	cmd := exec.Command("pgrep", "-f", "cursor")
	err := cmd.Run()
	return err == nil
}

// quitVSCode quits VS Code on Linux using pkill
func (e *VSCodeEditor) quitVSCode() error {
	// Try SIGTERM first (graceful)
	cmd := exec.Command("pkill", "-TERM", "-f", "code")
	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Fallback to SIGKILL (force)
	cmd = exec.Command("pkill", "-KILL", "-f", "code")
	return cmd.Run()
}

// isVSCodeRunning checks if VS Code is running on Linux
func (e *VSCodeEditor) isVSCodeRunning() bool {
	cmd := exec.Command("pgrep", "-f", "code")
	err := cmd.Run()
	return err == nil
}

// GetRunningProcesses returns a list of running editor processes on Linux
func GetRunningProcesses() ([]string, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "cursor") || strings.Contains(line, "code") {
			processes = append(processes, strings.TrimSpace(line))
		}
	}

	return processes, nil
}
