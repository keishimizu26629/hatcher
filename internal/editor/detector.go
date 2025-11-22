package editor

import (
	"os/exec"
	"sort"
	"strings"
)

// Editor represents an editor interface
type Editor interface {
	Name() string
	Command() string
	Priority() int
	IsInstalled() bool
	GetVersion() (string, error)
	Open(path string) error
	OpenInNewWindow(path string) error
	Quit() error
	IsRunning() bool
}

// EditorInfo contains information about an editor
type EditorInfo struct {
	Name        string
	Command     string
	VersionFlag string
	Priority    int
}

// Detector handles editor detection
type Detector struct {
	editors []EditorInfo
}

// NewDetector creates a new editor detector
func NewDetector() *Detector {
	return &Detector{
		editors: []EditorInfo{
			{
				Name:        "Cursor",
				Command:     "cursor",
				VersionFlag: "--version",
				Priority:    1, // Highest priority
			},
			{
				Name:        "VS Code",
				Command:     "code",
				VersionFlag: "--version",
				Priority:    2, // Second priority
			},
		},
	}
}

// DetectAvailable returns all available editors sorted by priority
func (d *Detector) DetectAvailable() []Editor {
	var available []Editor

	for _, info := range d.editors {
		editor := NewEditor(&info)
		if editor.IsInstalled() {
			available = append(available, editor)
		}
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(available, func(i, j int) bool {
		return available[i].Priority() < available[j].Priority()
	})

	return available
}

// GetBestEditor returns the best available editor (highest priority)
func (d *Detector) GetBestEditor() Editor {
	available := d.DetectAvailable()
	if len(available) > 0 {
		return available[0]
	}
	return nil
}

// GetEditorByName returns an editor by its command name
func (d *Detector) GetEditorByName(command string) Editor {
	for _, info := range d.editors {
		if info.Command == command {
			return NewEditor(&info)
		}
	}
	return nil
}

// BaseEditor provides common editor functionality
type BaseEditor struct {
	info *EditorInfo
}

// NewEditor creates a new editor instance
func NewEditor(info *EditorInfo) Editor {
	switch info.Command {
	case "cursor":
		return &CursorEditor{BaseEditor: BaseEditor{info: info}}
	case "code":
		return &VSCodeEditor{BaseEditor: BaseEditor{info: info}}
	default:
		return &BaseEditor{info: info}
	}
}

// Name returns the editor name
func (e *BaseEditor) Name() string {
	return e.info.Name
}

// Command returns the editor command
func (e *BaseEditor) Command() string {
	return e.info.Command
}

// Priority returns the editor priority
func (e *BaseEditor) Priority() int {
	return e.info.Priority
}

// IsInstalled checks if the editor is installed
func (e *BaseEditor) IsInstalled() bool {
	_, err := exec.LookPath(e.info.Command)
	return err == nil
}

// GetVersion returns the editor version
func (e *BaseEditor) GetVersion() (string, error) {
	cmd := exec.Command(e.info.Command, e.info.VersionFlag)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Open opens a path in the editor (default implementation)
func (e *BaseEditor) Open(path string) error {
	cmd := exec.Command(e.info.Command, path)
	return cmd.Start()
}

// OpenInNewWindow opens a path in a new editor window (default implementation)
func (e *BaseEditor) OpenInNewWindow(path string) error {
	cmd := exec.Command(e.info.Command, "--new-window", path)
	return cmd.Start()
}

// Quit quits the editor (default implementation)
func (e *BaseEditor) Quit() error {
	// Default implementation - not supported
	return nil
}

// IsRunning checks if the editor is running (default implementation)
func (e *BaseEditor) IsRunning() bool {
	// Default implementation - cannot determine
	return false
}

// CursorEditor implements Cursor-specific functionality
type CursorEditor struct {
	BaseEditor
}

// OpenInNewWindow opens a path in a new Cursor window
func (e *CursorEditor) OpenInNewWindow(path string) error {
	cmd := exec.Command(e.info.Command, "--new-window", path)
	return cmd.Start()
}

// Quit quits Cursor
func (e *CursorEditor) Quit() error {
	// Platform-specific implementation will be added later
	return e.quitCursor()
}

// IsRunning checks if Cursor is running
func (e *CursorEditor) IsRunning() bool {
	// Platform-specific implementation will be added later
	return e.isCursorRunning()
}

// VSCodeEditor implements VS Code-specific functionality
type VSCodeEditor struct {
	BaseEditor
}

// OpenInNewWindow opens a path in a new VS Code window
func (e *VSCodeEditor) OpenInNewWindow(path string) error {
	cmd := exec.Command(e.info.Command, "--new-window", path)
	return cmd.Start()
}

// Quit quits VS Code
func (e *VSCodeEditor) Quit() error {
	// Platform-specific implementation will be added later
	return e.quitVSCode()
}

// IsRunning checks if VS Code is running
func (e *VSCodeEditor) IsRunning() bool {
	// Platform-specific implementation will be added later
	return e.isVSCodeRunning()
}
