package worktree

import (
	"time"

	"github.com/keisukeshimizu/hatcher/internal/git"
)

// WorktreeInfo represents information about a worktree
type WorktreeInfo struct {
	Branch           string             `json:"branch"`
	Path             string             `json:"path"`
	Head             string             `json:"head"`
	Status           git.WorktreeStatus `json:"status"`
	Created          time.Time          `json:"created,omitempty"`
	IsMain           bool               `json:"isMain"`
	IsHatcherManaged bool               `json:"isHatcherManaged"`
	Editor           string             `json:"editor,omitempty"`
}

// WorktreeStatus represents the status of a worktree (alias for compatibility)
type WorktreeStatus = git.WorktreeStatus

// Status constants for compatibility
const (
	StatusClean   = git.StatusClean
	StatusDirty   = git.StatusDirty
	StatusActive  = git.StatusActive
	StatusUnknown = git.StatusUnknown
)
