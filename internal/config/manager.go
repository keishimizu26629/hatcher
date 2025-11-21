package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete Hatcher configuration
type Config struct {
	AutoCopy AutoCopyConfig `json:"autocopy" yaml:"autocopy"`
	Editor   EditorConfig   `json:"editor" yaml:"editor"`
	Global   GlobalConfig   `json:"global" yaml:"global"`
}

// AutoCopyConfig represents auto-copy configuration
type AutoCopyConfig struct {
	Version int            `json:"version" yaml:"version"`
	Items   []AutoCopyItem `json:"items" yaml:"items"`
	Files   []string       `json:"files,omitempty" yaml:"files,omitempty"` // For v1 compatibility
}

// AutoCopyItem represents a single item to be copied
type AutoCopyItem struct {
	Path       string `json:"path" yaml:"path"`
	Directory  *bool  `json:"directory,omitempty" yaml:"directory,omitempty"`
	Recursive  bool   `json:"recursive" yaml:"recursive"`
	RootOnly   bool   `json:"rootOnly" yaml:"rootOnly"`
	AutoDetect bool   `json:"autoDetect" yaml:"autoDetect"`
	Exclude    []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	Include    []string `json:"include,omitempty" yaml:"include,omitempty"`
}

// EditorConfig represents editor configuration
type EditorConfig struct {
	Preferred    string            `json:"preferred" yaml:"preferred"`
	AutoSwitch   bool              `json:"autoSwitch" yaml:"autoSwitch"`
	Commands     map[string]string `json:"commands,omitempty" yaml:"commands,omitempty"`
	WindowReuse  bool              `json:"windowReuse" yaml:"windowReuse"`
}

// GlobalConfig represents global settings
type GlobalConfig struct {
	Verbose     bool   `json:"verbose" yaml:"verbose"`
	OutputFormat string `json:"outputFormat" yaml:"outputFormat"`
	ColorOutput bool   `json:"colorOutput" yaml:"colorOutput"`
}

// Manager handles configuration loading, saving, and validation
type Manager struct {
	defaultConfig *Config
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		defaultConfig: getDefaultConfig(),
	}
}

// LoadConfig loads configuration from various sources with priority order
func (m *Manager) LoadConfig(projectPath string) (*Config, error) {
	config := m.defaultConfig.copy()

	// 1. Load global config
	if err := m.loadGlobalConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// 2. Load project config (if projectPath is provided)
	if projectPath != "" {
		if err := m.loadProjectConfig(config, projectPath); err != nil {
			return nil, fmt.Errorf("failed to load project config: %w", err)
		}
	}

	// 3. Apply environment variable overrides
	m.applyEnvironmentOverrides(config)

	// 4. Validate final configuration
	if errors := m.ValidateConfig(config); len(errors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return config, nil
}

// SaveConfig saves configuration to the specified location
func (m *Manager) SaveConfig(config *Config, projectPath string, global bool) error {
	var configPath string
	var data []byte
	var err error

	if global {
		// Save as global YAML config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".hatcher")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configPath = filepath.Join(configDir, "config.yaml")
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	} else {
		// Save as project JSON config (auto-copy only)
		if projectPath == "" {
			return fmt.Errorf("project path is required for project config")
		}

		configPath = filepath.Join(projectPath, ".hatcher-auto-copy.json")
		data, err = json.MarshalIndent(config.AutoCopy, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration and returns any errors
func (m *Manager) ValidateConfig(config *Config) []string {
	var errors []string

	// Validate AutoCopy configuration
	if config.AutoCopy.Version < 1 || config.AutoCopy.Version > 2 {
		errors = append(errors, fmt.Sprintf("unsupported autocopy version: %d", config.AutoCopy.Version))
	}

	for i, item := range config.AutoCopy.Items {
		if item.Path == "" {
			errors = append(errors, fmt.Sprintf("autocopy item %d has empty path", i))
		}

		if strings.Contains(item.Path, "..") {
			errors = append(errors, fmt.Sprintf("autocopy item %d contains invalid path: %s", i, item.Path))
		}
	}

	// Validate Editor configuration
	if config.Editor.Preferred != "" {
		validEditors := []string{"cursor", "code", "vim", "nano", ""}
		valid := false
		for _, editor := range validEditors {
			if config.Editor.Preferred == editor {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("unsupported editor: %s", config.Editor.Preferred))
		}
	}

	// Validate Global configuration
	if config.Global.OutputFormat != "" {
		validFormats := []string{"table", "json", "yaml", "simple"}
		valid := false
		for _, format := range validFormats {
			if config.Global.OutputFormat == format {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("unsupported output format: %s", config.Global.OutputFormat))
		}
	}

	return errors
}

// MigrateConfig migrates configuration from older versions
func (m *Manager) MigrateConfig(rawConfig map[string]interface{}) (*Config, error) {
	config := m.defaultConfig.copy()

	version, ok := rawConfig["version"].(float64)
	if !ok {
		version = 1 // Default to v1 if no version specified
	}

	switch int(version) {
	case 1:
		// Migrate from v1 to v2
		if files, ok := rawConfig["files"].([]interface{}); ok {
			config.AutoCopy.Version = 2
			config.AutoCopy.Items = make([]AutoCopyItem, 0, len(files))

			for _, file := range files {
				if filePath, ok := file.(string); ok {
					item := AutoCopyItem{
						Path:       filePath,
						AutoDetect: true,
					}

					// Auto-detect if it's a directory
					if strings.HasSuffix(filePath, "/") {
						item.Directory = boolPtr(true)
						item.Recursive = true
					} else {
						item.Directory = boolPtr(false)
					}

					config.AutoCopy.Items = append(config.AutoCopy.Items, item)
				}
			}
		}

	case 2:
		// Already v2, just parse normally
		if err := m.parseV2Config(config, rawConfig); err != nil {
			return nil, fmt.Errorf("failed to parse v2 config: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported config version: %d", int(version))
	}

	return config, nil
}

// GetConfigPaths returns all possible configuration file paths in priority order
func (m *Manager) GetConfigPaths(projectPath string) []string {
	var paths []string

	if projectPath != "" {
		// Project-specific configs
		paths = append(paths,
			filepath.Join(projectPath, ".hatcher-auto-copy.json"),
			filepath.Join(projectPath, ".hatcher-auto-copy.yaml"),
			filepath.Join(projectPath, ".hatcher", "config.json"),
			filepath.Join(projectPath, ".hatcher", "config.yaml"),
		)
	}

	// Global configs
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(homeDir, ".hatcher", "config.json"),
			filepath.Join(homeDir, ".hatcher", "config.yaml"),
		)
	}

	return paths
}

// loadGlobalConfig loads global configuration
func (m *Manager) loadGlobalConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Skip global config if home directory is not available
	}

	configPaths := []string{
		filepath.Join(homeDir, ".hatcher", "config.yaml"),
		filepath.Join(homeDir, ".hatcher", "config.json"),
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		var rawConfig map[string]interface{}
		if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
			err = yaml.Unmarshal(data, &rawConfig)
		} else {
			err = json.Unmarshal(data, &rawConfig)
		}

		if err != nil {
			continue
		}

		// Merge global config
		if err := m.mergeConfig(config, rawConfig); err != nil {
			return err
		}

		break // Use first found config
	}

	return nil
}

// loadProjectConfig loads project-specific configuration
func (m *Manager) loadProjectConfig(config *Config, projectPath string) error {
	configPaths := []string{
		filepath.Join(projectPath, ".hatcher-auto-copy.json"),
		filepath.Join(projectPath, ".hatcher-auto-copy.yaml"),
		filepath.Join(projectPath, ".hatcher", "config.json"),
		filepath.Join(projectPath, ".hatcher", "config.yaml"),
	}

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		var rawConfig map[string]interface{}
		if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
			err = yaml.Unmarshal(data, &rawConfig)
		} else {
			err = json.Unmarshal(data, &rawConfig)
		}

		if err != nil {
			continue
		}

		// Check if this is an old format auto-copy config
		if _, hasVersion := rawConfig["version"]; hasVersion {
			if _, hasItems := rawConfig["items"]; hasItems || rawConfig["files"] != nil {
				// This is an auto-copy specific config, migrate it
				migratedConfig, err := m.MigrateConfig(rawConfig)
				if err != nil {
					return err
				}
				config.AutoCopy = migratedConfig.AutoCopy
				break
			}
		}

		// Merge project config
		if err := m.mergeConfig(config, rawConfig); err != nil {
			return err
		}

		break // Use first found config
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (m *Manager) applyEnvironmentOverrides(config *Config) {
	if editor := os.Getenv("HATCHER_EDITOR"); editor != "" {
		config.Editor.Preferred = editor
	}

	if verbose := os.Getenv("HATCHER_VERBOSE"); verbose != "" {
		if v, err := strconv.ParseBool(verbose); err == nil {
			config.Global.Verbose = v
		}
	}

	if format := os.Getenv("HATCHER_OUTPUT_FORMAT"); format != "" {
		config.Global.OutputFormat = format
	}

	if color := os.Getenv("HATCHER_COLOR"); color != "" {
		if v, err := strconv.ParseBool(color); err == nil {
			config.Global.ColorOutput = v
		}
	}
}

// mergeConfig merges raw configuration into the config object
func (m *Manager) mergeConfig(config *Config, rawConfig map[string]interface{}) error {
	// This is a simplified merge - in a real implementation,
	// you'd want more sophisticated merging logic

	if autocopy, ok := rawConfig["autocopy"].(map[string]interface{}); ok {
		if err := m.parseAutoCopyConfig(&config.AutoCopy, autocopy); err != nil {
			return err
		}
	}

	if editor, ok := rawConfig["editor"].(map[string]interface{}); ok {
		if err := m.parseEditorConfig(&config.Editor, editor); err != nil {
			return err
		}
	}

	if global, ok := rawConfig["global"].(map[string]interface{}); ok {
		if err := m.parseGlobalConfig(&config.Global, global); err != nil {
			return err
		}
	}

	return nil
}

// parseV2Config parses v2 configuration format
func (m *Manager) parseV2Config(config *Config, rawConfig map[string]interface{}) error {
	return m.mergeConfig(config, rawConfig)
}

// parseAutoCopyConfig parses auto-copy configuration
func (m *Manager) parseAutoCopyConfig(config *AutoCopyConfig, raw map[string]interface{}) error {
	if version, ok := raw["version"].(float64); ok {
		config.Version = int(version)
	}

	if items, ok := raw["items"].([]interface{}); ok {
		config.Items = make([]AutoCopyItem, 0, len(items))
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				var autoCopyItem AutoCopyItem
				if err := m.parseAutoCopyItem(&autoCopyItem, itemMap); err != nil {
					return err
				}
				config.Items = append(config.Items, autoCopyItem)
			}
		}
	}

	return nil
}

// parseAutoCopyItem parses a single auto-copy item
func (m *Manager) parseAutoCopyItem(item *AutoCopyItem, raw map[string]interface{}) error {
	if path, ok := raw["path"].(string); ok {
		item.Path = path
	}

	if directory, ok := raw["directory"].(bool); ok {
		item.Directory = &directory
	}

	if recursive, ok := raw["recursive"].(bool); ok {
		item.Recursive = recursive
	}

	if rootOnly, ok := raw["rootOnly"].(bool); ok {
		item.RootOnly = rootOnly
	}

	if autoDetect, ok := raw["autoDetect"].(bool); ok {
		item.AutoDetect = autoDetect
	}

	return nil
}

// parseEditorConfig parses editor configuration
func (m *Manager) parseEditorConfig(config *EditorConfig, raw map[string]interface{}) error {
	if preferred, ok := raw["preferred"].(string); ok {
		config.Preferred = preferred
	}

	if autoSwitch, ok := raw["autoSwitch"].(bool); ok {
		config.AutoSwitch = autoSwitch
	}

	if windowReuse, ok := raw["windowReuse"].(bool); ok {
		config.WindowReuse = windowReuse
	}

	return nil
}

// parseGlobalConfig parses global configuration
func (m *Manager) parseGlobalConfig(config *GlobalConfig, raw map[string]interface{}) error {
	if verbose, ok := raw["verbose"].(bool); ok {
		config.Verbose = verbose
	}

	if outputFormat, ok := raw["outputFormat"].(string); ok {
		config.OutputFormat = outputFormat
	}

	if colorOutput, ok := raw["colorOutput"].(bool); ok {
		config.ColorOutput = colorOutput
	}

	return nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		AutoCopy: AutoCopyConfig{
			Version: 2,
			Items: []AutoCopyItem{
				{
					Path:       ".ai/",
					Directory:  boolPtr(true),
					Recursive:  true,
					AutoDetect: true,
				},
				{
					Path:       ".cursorrules",
					Directory:  boolPtr(false),
					AutoDetect: true,
				},
				{
					Path:       ".clinerules",
					Directory:  boolPtr(false),
					AutoDetect: true,
				},
				{
					Path:       "CLAUDE.md",
					Directory:  boolPtr(false),
					AutoDetect: true,
				},
			},
		},
		Editor: EditorConfig{
			Preferred:   "cursor",
			AutoSwitch:  false,
			WindowReuse: true,
		},
		Global: GlobalConfig{
			Verbose:      false,
			OutputFormat: "table",
			ColorOutput:  true,
		},
	}
}

// copy creates a deep copy of the configuration
func (c *Config) copy() *Config {
	newConfig := &Config{
		AutoCopy: AutoCopyConfig{
			Version: c.AutoCopy.Version,
			Items:   make([]AutoCopyItem, len(c.AutoCopy.Items)),
			Files:   make([]string, len(c.AutoCopy.Files)),
		},
		Editor: c.Editor,
		Global: c.Global,
	}

	copy(newConfig.AutoCopy.Items, c.AutoCopy.Items)
	copy(newConfig.AutoCopy.Files, c.AutoCopy.Files)

	// Deep copy directory pointers
	for i := range newConfig.AutoCopy.Items {
		if c.AutoCopy.Items[i].Directory != nil {
			newConfig.AutoCopy.Items[i].Directory = boolPtr(*c.AutoCopy.Items[i].Directory)
		}
	}

	return newConfig
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
