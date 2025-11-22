package autocopy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AutoCopyConfig represents the configuration for automatic file copying
type AutoCopyConfig struct {
	Version int            `json:"version"`
	Items   []AutoCopyItem `json:"items"`
	Files   []string       `json:"files,omitempty"` // Legacy format support
}

// AutoCopyItem represents a single item to be copied
type AutoCopyItem struct {
	Path       string   `json:"path"`
	Directory  *bool    `json:"directory,omitempty"`
	Recursive  bool     `json:"recursive"`
	RootOnly   bool     `json:"rootOnly"`
	AutoDetect bool     `json:"autoDetect"`
	UseGlob    bool     `json:"useGlob"`
	Exclude    []string `json:"exclude,omitempty"`
	Include    []string `json:"include,omitempty"`
}

// IsDirectory returns true if the item should be treated as a directory
func (item *AutoCopyItem) IsDirectory() bool {
	// If autoDetect is enabled, return false as default (will be determined at runtime)
	if item.AutoDetect {
		return false
	}

	// If directory is explicitly set, use that value
	if item.Directory != nil {
		return *item.Directory
	}

	// Auto-detect based on path ending with '/'
	return strings.HasSuffix(item.Path, "/")
}

// IsGlobPattern returns true if the path contains glob pattern characters
func (item *AutoCopyItem) IsGlobPattern() bool {
	path := item.Path
	return strings.ContainsAny(path, "*?[")
}

// LoadAutoCopyConfig loads configuration from the first available file in the given paths
func LoadAutoCopyConfig(paths []string) (*AutoCopyConfig, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var config AutoCopyConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
			}

			return &config, nil
		}
	}

	// Return empty config if no file found
	return &AutoCopyConfig{}, nil
}

// ValidateAutoCopyConfig validates the configuration
func ValidateAutoCopyConfig(config *AutoCopyConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate legacy format
	if config.Version == 0 && len(config.Files) > 0 {
		for _, file := range config.Files {
			if err := validatePath(file); err != nil {
				return err
			}
		}
		return nil
	}

	// Validate new format
	for i, item := range config.Items {
		if err := validateAutoCopyItem(item, i); err != nil {
			return err
		}
	}

	return nil
}

// validatePath validates a file path for security
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for dangerous path patterns
	dangerous := []string{"..", "//", "\\\\"}
	for _, pattern := range dangerous {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("dangerous path pattern detected: %s", pattern)
		}
	}

	return nil
}

// validateAutoCopyItem validates a single auto-copy item
func validateAutoCopyItem(item AutoCopyItem, index int) error {
	if err := validatePath(item.Path); err != nil {
		return fmt.Errorf("item %d: %w", index, err)
	}

	// Check for conflicting options
	if item.Directory != nil && item.AutoDetect {
		return fmt.Errorf("item %d: cannot use both directory and autoDetect options", index)
	}

	return nil
}
