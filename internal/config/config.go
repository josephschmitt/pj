package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

// MarkerConfig represents a single marker with optional icon and priority (new format)
type MarkerConfig struct {
	Marker      string `yaml:"marker"`
	Icon        string `yaml:"icon,omitempty"`
	Priority    int    `yaml:"priority,omitempty"`
	HasIcon     bool   `yaml:"-"` // True if icon field was explicitly set in config
	HasPriority bool   `yaml:"-"` // True if priority field was explicitly set in config
}

// MarkerList handles unmarshaling both old format ([]string) and new format ([]MarkerConfig)
type MarkerList []MarkerConfig

// UnmarshalYAML implements custom unmarshaling to support both formats
func (m *MarkerList) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as sequence
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("markers must be a list")
	}

	*m = make([]MarkerConfig, 0, len(value.Content))

	for _, item := range value.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			// Old format: simple string
			*m = append(*m, MarkerConfig{Marker: item.Value})
		case yaml.MappingNode:
			// New format: object with marker and optional icon
			var mc MarkerConfig
			if err := item.Decode(&mc); err != nil {
				return fmt.Errorf("invalid marker config: %w", err)
			}
			if mc.Marker == "" {
				return fmt.Errorf("marker config must have a 'marker' field")
			}
			// Check if icon/priority fields were explicitly present
			for i := 0; i < len(item.Content); i += 2 {
				switch item.Content[i].Value {
				case "icon":
					mc.HasIcon = true
				case "priority":
					mc.HasPriority = true
				}
			}
			*m = append(*m, mc)
		default:
			return fmt.Errorf("marker must be a string or object, got %v", item.Kind)
		}
	}

	return nil
}

// Config holds the application configuration
type Config struct {
	SearchPaths []string          `yaml:"search_paths"`
	RawMarkers  MarkerList        `yaml:"markers"`
	Markers     []string          `yaml:"-"` // Derived from RawMarkers for internal use
	MaxDepth    int               `yaml:"max_depth"`
	Excludes    []string          `yaml:"excludes"`
	CacheTTL    int               `yaml:"cache_ttl"` // seconds
	NoIgnore    bool              `yaml:"no_ignore"` // Don't respect .gitignore and .ignore files
	Nested      bool              `yaml:"nested"`    // Continue discovery inside projects
	// Deprecated: Use the new markers format with icon field instead.
	// This field is kept for backward compatibility.
	Icons map[string]string `yaml:"icons,omitempty"`

	// Priorities maps marker names to their priority values (derived from RawMarkers)
	Priorities map[string]int `yaml:"-"`

	// Internal flags for detecting format conflicts
	hasNewFormatIcons bool
	hasOldFormatIcons bool
}

// GetIcons returns the merged icon map for use by the application.
// This is the preferred way to access icons programmatically.
func (c *Config) GetIcons() map[string]string {
	return c.Icons
}

// GetPriorities returns a copy of the priorities map for use by the application.
func (c *Config) GetPriorities() map[string]int {
	m := make(map[string]int)
	for k, v := range c.Priorities {
		m[k] = v
	}
	return m
}

// CLI interface for merging flags
type CLIFlags interface {
	GetPaths() []string
	GetMarkers() []string
	GetExcludes() []string
	GetMaxDepth() int
}

// Load loads configuration from file or returns defaults
func Load(configPath string) (*Config, error) {
	return LoadWithVerbose(configPath, false)
}

// LoadWithVerbose loads configuration and optionally emits deprecation warnings
func LoadWithVerbose(configPath string, verbose bool) (*Config, error) {
	cfg := defaults()
	cfg.processMarkers() // Build default icons and priorities from RawMarkers

	// Save defaults before YAML unmarshaling overwrites them
	defaultIcons := cfg.Icons
	defaultPriorities := cfg.Priorities
	defaultRawMarkers := cfg.RawMarkers

	if configPath == "" {
		configPath = defaultConfigPath()
	}

	if len(configPath) > 0 && configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, use defaults (already processed)
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	// Reset fields before unmarshal so we can detect what YAML provides
	cfg.Icons = nil
	cfg.RawMarkers = nil

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Merge YAML markers with defaults (YAML takes precedence for icons)
	yamlHadMarkers := cfg.RawMarkers != nil
	cfg.RawMarkers = mergeMarkers(defaultRawMarkers, cfg.RawMarkers)

	cfg.processMarkersWithDefaults(defaultIcons, defaultPriorities, yamlHadMarkers)
	cfg.emitDeprecationWarnings(verbose)

	return cfg, nil
}

// processMarkers builds the Markers slice, Icons map, and Priorities map from RawMarkers
// Used for processing defaults (doesn't set deprecation flags)
func (c *Config) processMarkers() {
	c.Markers = make([]string, len(c.RawMarkers))
	for i, mc := range c.RawMarkers {
		c.Markers[i] = mc.Marker
	}
	// Build icons from RawMarkers for defaults
	c.Icons = make(map[string]string)
	for _, mc := range c.RawMarkers {
		if mc.HasIcon {
			c.Icons[mc.Marker] = mc.Icon
		}
	}
	// Build priorities from RawMarkers for defaults
	c.Priorities = make(map[string]int)
	for _, mc := range c.RawMarkers {
		if mc.HasPriority {
			c.Priorities[mc.Marker] = mc.Priority
		}
	}
}

// processMarkersWithDefaults builds Markers/Icons/Priorities, merging with defaults
// yamlHadMarkers indicates whether the YAML config had a markers field
func (c *Config) processMarkersWithDefaults(defaultIcons map[string]string, defaultPriorities map[string]int, yamlHadMarkers bool) {
	c.Markers = make([]string, len(c.RawMarkers))
	newIcons := make(map[string]string)
	newPriorities := make(map[string]int)
	// Track markers that explicitly set icons/priorities (even to empty/zero)
	explicitIcons := make(map[string]bool)
	explicitPriorities := make(map[string]bool)

	for i, mc := range c.RawMarkers {
		c.Markers[i] = mc.Marker
		// Only consider this a "new format" if YAML actually had a markers field
		if yamlHadMarkers {
			if mc.HasIcon {
				newIcons[mc.Marker] = mc.Icon
				explicitIcons[mc.Marker] = true
				c.hasNewFormatIcons = true
			}
			if mc.HasPriority {
				newPriorities[mc.Marker] = mc.Priority
				explicitPriorities[mc.Marker] = true
			}
		}
	}

	// Check if old Icons field was populated from YAML
	if len(c.Icons) > 0 {
		c.hasOldFormatIcons = true
	}

	// Merge icons: defaults -> old format -> new format (later wins)
	// But skip defaults for markers that explicitly set icons in new format
	finalIcons := make(map[string]string)
	for k, v := range defaultIcons {
		if !explicitIcons[k] {
			finalIcons[k] = v
		}
	}
	for k, v := range c.Icons {
		finalIcons[k] = v
	}
	for k, v := range newIcons {
		finalIcons[k] = v
	}
	c.Icons = finalIcons

	// Merge priorities: defaults -> new format (later wins)
	finalPriorities := make(map[string]int)
	for k, v := range defaultPriorities {
		if !explicitPriorities[k] {
			finalPriorities[k] = v
		}
	}
	for k, v := range newPriorities {
		finalPriorities[k] = v
	}
	c.Priorities = finalPriorities
}

// mergeMarkers combines default markers with YAML markers
// YAML markers override defaults for the same marker name
func mergeMarkers(defaults, yaml MarkerList) MarkerList {
	if yaml == nil {
		return defaults
	}

	// Build a map of YAML markers for quick lookup
	yamlMap := make(map[string]MarkerConfig)
	for _, mc := range yaml {
		yamlMap[mc.Marker] = mc
	}

	// Start with defaults, override with YAML where present
	result := make(MarkerList, 0, len(defaults)+len(yaml))
	seen := make(map[string]bool)

	for _, mc := range defaults {
		if yamlMc, exists := yamlMap[mc.Marker]; exists {
			// YAML overrides this default
			result = append(result, yamlMc)
		} else {
			// Keep default
			result = append(result, mc)
		}
		seen[mc.Marker] = true
	}

	// Add any YAML markers that weren't in defaults
	for _, mc := range yaml {
		if !seen[mc.Marker] {
			result = append(result, mc)
		}
	}

	return result
}

// emitDeprecationWarnings outputs warnings about deprecated configuration usage
func (c *Config) emitDeprecationWarnings(verbose bool) {
	if !verbose {
		return
	}

	if c.hasOldFormatIcons && c.hasNewFormatIcons {
		fmt.Fprintf(os.Stderr, "Warning: Both new 'markers' format with icons and deprecated 'icons' field are present.\n")
		fmt.Fprintf(os.Stderr, "The new format takes precedence. Consider removing the 'icons' field from your configuration.\n")
	} else if c.hasOldFormatIcons && !c.hasNewFormatIcons {
		fmt.Fprintf(os.Stderr, "Warning: The 'icons' field in configuration is deprecated.\n")
		fmt.Fprintf(os.Stderr, "Consider migrating to the new format:\n")
		fmt.Fprintf(os.Stderr, "  markers:\n")
		fmt.Fprintf(os.Stderr, "    - marker: .git\n")
		fmt.Fprintf(os.Stderr, "      icon: \"\"\n")
	}
}

// MergeFlags merges CLI flags into the config (CLI flags take precedence)
func (c *Config) MergeFlags(cli interface{}) error {
	v := reflect.ValueOf(cli)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if pathField := v.FieldByName("Path"); pathField.IsValid() && pathField.Kind() == reflect.Slice {
		if pathField.Len() > 0 {
			for i := 0; i < pathField.Len(); i++ {
				path := pathField.Index(i).String()
				c.SearchPaths = append(c.SearchPaths, path)
			}
		}
	}

	if markerField := v.FieldByName("Marker"); markerField.IsValid() && markerField.Kind() == reflect.Slice {
		if markerField.Len() > 0 {
			for i := 0; i < markerField.Len(); i++ {
				marker := markerField.Index(i).String()
				c.Markers = append(c.Markers, marker)
			}
		}
	}

	if excludeField := v.FieldByName("Exclude"); excludeField.IsValid() && excludeField.Kind() == reflect.Slice {
		if excludeField.Len() > 0 {
			for i := 0; i < excludeField.Len(); i++ {
				exclude := excludeField.Index(i).String()
				c.Excludes = append(c.Excludes, exclude)
			}
		}
	}

	if maxDepthField := v.FieldByName("MaxDepth"); maxDepthField.IsValid() && maxDepthField.Kind() == reflect.Int {
		if maxDepth := int(maxDepthField.Int()); maxDepth > 0 {
			c.MaxDepth = maxDepth
		}
	}

	if noIgnoreField := v.FieldByName("NoIgnore"); noIgnoreField.IsValid() && noIgnoreField.Kind() == reflect.Bool {
		c.NoIgnore = noIgnoreField.Bool()
	}

	if noNestedField := v.FieldByName("NoNested"); noNestedField.IsValid() && noNestedField.Kind() == reflect.Bool {
		if noNestedField.Bool() {
			c.Nested = false
		}
	}

	return nil
}

// defaults returns the default configuration
func defaults() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		SearchPaths: []string{
			filepath.Join(home, "projects"),
			filepath.Join(home, "code"),
			filepath.Join(home, "development"),
		},
		RawMarkers: MarkerList{
			{Marker: ".git", Icon: "\ue65d", HasIcon: true, Priority: 1, HasPriority: true},
			{Marker: "go.mod", Icon: "\U000f07d3", HasIcon: true, Priority: 10, HasPriority: true},
			{Marker: "package.json", Icon: "\U000f0399", HasIcon: true, Priority: 10, HasPriority: true},
			{Marker: "Cargo.toml", Icon: "\ue68b", HasIcon: true, Priority: 10, HasPriority: true},
			{Marker: "pyproject.toml", Icon: "\ue606", HasIcon: true, Priority: 10, HasPriority: true},
			{Marker: "Makefile", Icon: "\ue673", HasIcon: true, Priority: 1, HasPriority: true},
			{Marker: "flake.nix", Icon: "\ue843", HasIcon: true, Priority: 10, HasPriority: true},
			{Marker: ".vscode", Icon: "\U000f0a1e", HasIcon: true, Priority: 5, HasPriority: true},
			{Marker: ".idea", Icon: "\ue7b5", HasIcon: true, Priority: 5, HasPriority: true},
			{Marker: ".fleet", Priority: 5, HasPriority: true},
			{Marker: ".project", Icon: "\ue79e", HasIcon: true, Priority: 5, HasPriority: true},
			{Marker: ".zed", Priority: 5, HasPriority: true},
			{Marker: "Dockerfile", Icon: "\ue7b0", HasIcon: true, Priority: 7, HasPriority: true},
		},
		MaxDepth: 3,
		Excludes: []string{
			"node_modules",
			".terraform",
			"vendor",
			".git",
			"target",
			"dist",
			"build",
		},
		CacheTTL: 300, // 5 minutes
		Nested:   true,
		Icons:    make(map[string]string),
	}
}

// defaultConfigPath returns the default config file path using XDG_CONFIG_HOME
func defaultConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "pj", "config.yaml")
}
