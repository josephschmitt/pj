package config

import (
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	SearchPaths []string          `yaml:"search_paths"`
	Markers     []string          `yaml:"markers"`
	MaxDepth    int               `yaml:"max_depth"`
	Excludes    []string          `yaml:"excludes"`
	CacheTTL    int               `yaml:"cache_ttl"` // seconds
	Icons       map[string]string `yaml:"icons"`
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
	cfg := defaults()

	// Determine config file path
	if configPath == "" {
		configPath = defaultConfigPath()
	}

	// Expand home directory
	if len(configPath) > 0 && configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	// Try to load config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, use defaults
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// MergeFlags merges CLI flags into the config (CLI flags take precedence)
func (c *Config) MergeFlags(cli interface{}) error {
	// Use reflection to access fields
	v := reflect.ValueOf(cli)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Get Path field
	if pathField := v.FieldByName("Path"); pathField.IsValid() && pathField.Kind() == reflect.Slice {
		if pathField.Len() > 0 {
			for i := 0; i < pathField.Len(); i++ {
				path := pathField.Index(i).String()
				c.SearchPaths = append(c.SearchPaths, path)
			}
		}
	}

	// Get Marker field
	if markerField := v.FieldByName("Marker"); markerField.IsValid() && markerField.Kind() == reflect.Slice {
		if markerField.Len() > 0 {
			for i := 0; i < markerField.Len(); i++ {
				marker := markerField.Index(i).String()
				c.Markers = append(c.Markers, marker)
			}
		}
	}

	// Get Exclude field
	if excludeField := v.FieldByName("Exclude"); excludeField.IsValid() && excludeField.Kind() == reflect.Slice {
		if excludeField.Len() > 0 {
			for i := 0; i < excludeField.Len(); i++ {
				exclude := excludeField.Index(i).String()
				c.Excludes = append(c.Excludes, exclude)
			}
		}
	}

	// Get MaxDepth field
	if maxDepthField := v.FieldByName("MaxDepth"); maxDepthField.IsValid() && maxDepthField.Kind() == reflect.Int {
		if maxDepth := int(maxDepthField.Int()); maxDepth > 0 {
			c.MaxDepth = maxDepth
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
		Markers: []string{
			".git",
			"go.mod",
			"package.json",
			"Cargo.toml",
			"pyproject.toml",
			"Makefile",
			"flake.nix",
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
		Icons: map[string]string{
			".git":           "",
			"go.mod":         "󰟓",
			"package.json":   "󰎙",
			"Cargo.toml":     "",
			"pyproject.toml": "",
			"Makefile":       "",
			"flake.nix":      "",
		},
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
