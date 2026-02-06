package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()
	cfg.processMarkers() // Process RawMarkers to populate Markers and Icons

	// Verify default values
	if cfg == nil {
		t.Fatal("defaults() returned nil")
	}

	// Check search paths
	if len(cfg.SearchPaths) == 0 {
		t.Error("defaults() should have search paths")
	}

	home, _ := os.UserHomeDir()
	expectedPaths := []string{
		filepath.Join(home, "projects"),
		filepath.Join(home, "code"),
		filepath.Join(home, "development"),
	}
	if len(cfg.SearchPaths) != len(expectedPaths) {
		t.Errorf("SearchPaths length = %d, want %d", len(cfg.SearchPaths), len(expectedPaths))
	}
	for i, path := range expectedPaths {
		if cfg.SearchPaths[i] != path {
			t.Errorf("SearchPaths[%d] = %q, want %q", i, cfg.SearchPaths[i], path)
		}
	}

	// Check RawMarkers (new format)
	expectedMarkers := []string{
		".git",
		"go.mod",
		"package.json",
		"Cargo.toml",
		"pyproject.toml",
		"Makefile",
		"flake.nix",
		".vscode",
		".idea",
		".fleet",
		".project",
		".zed",
		"Dockerfile",
	}
	if len(cfg.RawMarkers) != len(expectedMarkers) {
		t.Errorf("RawMarkers length = %d, want %d", len(cfg.RawMarkers), len(expectedMarkers))
	}
	for i, expected := range expectedMarkers {
		if cfg.RawMarkers[i].Marker != expected {
			t.Errorf("RawMarkers[%d].Marker = %q, want %q", i, cfg.RawMarkers[i].Marker, expected)
		}
	}

	// Check that Markers slice is populated after processMarkers
	if len(cfg.Markers) != len(expectedMarkers) {
		t.Errorf("Markers length = %d, want %d", len(cfg.Markers), len(expectedMarkers))
	}

	// Check max depth
	if cfg.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", cfg.MaxDepth)
	}

	// Check excludes
	if len(cfg.Excludes) == 0 {
		t.Error("defaults() should have excludes")
	}

	// Check cache TTL
	if cfg.CacheTTL != 300 {
		t.Errorf("CacheTTL = %d, want 300", cfg.CacheTTL)
	}

	// Check nested defaults to true
	if !cfg.Nested {
		t.Error("Nested should default to true")
	}

	// Check icons map is populated from RawMarkers
	if len(cfg.Icons) == 0 {
		t.Error("defaults() should have icons after processMarkers")
	}
	// Verify specific icon exists
	if _, ok := cfg.Icons[".git"]; !ok {
		t.Error("Icons should include .git icon")
	}

	// Check priorities map is populated from RawMarkers
	if len(cfg.Priorities) == 0 {
		t.Error("defaults() should have priorities after processMarkers")
	}
	// Verify specific priorities
	expectedPriorities := map[string]int{
		".git":           1,
		"go.mod":         10,
		"package.json":   10,
		"Cargo.toml":     10,
		"pyproject.toml": 10,
		"Makefile":       1,
		"flake.nix":      10,
		".vscode":        5,
		".idea":          5,
		".fleet":         5,
		".project":       5,
		".zed":           5,
		"Dockerfile":     7,
	}
	for marker, expectedPriority := range expectedPriorities {
		if cfg.Priorities[marker] != expectedPriority {
			t.Errorf("Priorities[%q] = %d, want %d", marker, cfg.Priorities[marker], expectedPriority)
		}
	}
}

func TestDefaultConfigPath(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		path := defaultConfigPath()
		expected := filepath.Join(tmpDir, "pj", "config.yaml")
		if path != expected {
			t.Errorf("defaultConfigPath() = %q, want %q", path, expected)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		path := defaultConfigPath()
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "pj", "config.yaml")
		if path != expected {
			t.Errorf("defaultConfigPath() = %q, want %q", path, expected)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("no config file returns defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

		cfg, err := Load(nonExistentPath)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}

		// Should have default values
		if len(cfg.SearchPaths) == 0 {
			t.Error("Config should have default search paths")
		}
	})

	t.Run("valid YAML file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `search_paths:
  - /custom/path1
  - /custom/path2
markers:
  - .git
  - custom.marker
max_depth: 5
excludes:
  - node_modules
cache_ttl: 600
icons:
  .git: "custom-icon"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}

		if len(cfg.SearchPaths) != 2 {
			t.Errorf("SearchPaths length = %d, want 2", len(cfg.SearchPaths))
		}
		if cfg.SearchPaths[0] != "/custom/path1" {
			t.Errorf("SearchPaths[0] = %q, want %q", cfg.SearchPaths[0], "/custom/path1")
		}
		if cfg.MaxDepth != 5 {
			t.Errorf("MaxDepth = %d, want 5", cfg.MaxDepth)
		}
		if cfg.CacheTTL != 600 {
			t.Errorf("CacheTTL = %d, want 600", cfg.CacheTTL)
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		invalidYAML := `search_paths:
  - invalid
    - broken: yaml
`
		if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() with invalid YAML should return error")
		}
	})

	t.Run("tilde expansion", func(t *testing.T) {
		// Create a config file in temp dir for testing
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `search_paths:
  - ~/projects
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Test loading with tilde in path
		home, _ := os.UserHomeDir()
		tildeConfigPath := "~/" + configPath[len(home)+1:]

		cfg, err := Load(tildeConfigPath)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
	})

	t.Run("permission error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Create file with no read permissions
		if err := os.WriteFile(configPath, []byte("test"), 0000); err != nil {
			t.Fatal(err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() with unreadable file should return error")
		}

		// Clean up
		if err := os.Chmod(configPath, 0644); err != nil {
			t.Logf("Warning: failed to restore permissions: %v", err)
		}
	})
}

func TestMergeFlags(t *testing.T) {
	t.Run("empty slices don't override", func(t *testing.T) {
		cfg := &Config{
			SearchPaths: []string{"/original/path"},
			Markers:     []string{".git"},
			Excludes:    []string{"node_modules"},
		}

		flags := struct {
			Path    []string
			Marker  []string
			Exclude []string
		}{
			Path:    []string{},
			Marker:  []string{},
			Exclude: []string{},
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		// Original values should be preserved
		if len(cfg.SearchPaths) != 1 || cfg.SearchPaths[0] != "/original/path" {
			t.Error("Empty Path slice should not override SearchPaths")
		}
		if len(cfg.Markers) != 1 || cfg.Markers[0] != ".git" {
			t.Error("Empty Marker slice should not override Markers")
		}
		if len(cfg.Excludes) != 1 || cfg.Excludes[0] != "node_modules" {
			t.Error("Empty Exclude slice should not override Excludes")
		}
	})

	t.Run("zero MaxDepth doesn't override", func(t *testing.T) {
		cfg := &Config{
			MaxDepth: 5,
		}

		flags := struct {
			MaxDepth int
		}{
			MaxDepth: 0,
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		// Original MaxDepth should be preserved
		if cfg.MaxDepth != 5 {
			t.Errorf("MaxDepth = %d, want 5 (zero should not override)", cfg.MaxDepth)
		}
	})

	t.Run("non-empty slices append", func(t *testing.T) {
		cfg := &Config{
			SearchPaths: []string{"/original/path"},
			Markers:     []string{".git"},
			Excludes:    []string{"node_modules"},
		}

		flags := struct {
			Path    []string
			Marker  []string
			Exclude []string
		}{
			Path:    []string{"/new/path"},
			Marker:  []string{"go.mod"},
			Exclude: []string{"vendor"},
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		// Values should be appended
		if len(cfg.SearchPaths) != 2 {
			t.Errorf("SearchPaths length = %d, want 2", len(cfg.SearchPaths))
		}
		if cfg.SearchPaths[1] != "/new/path" {
			t.Errorf("SearchPaths[1] = %q, want %q", cfg.SearchPaths[1], "/new/path")
		}

		if len(cfg.Markers) != 2 {
			t.Errorf("Markers length = %d, want 2", len(cfg.Markers))
		}
		if cfg.Markers[1] != "go.mod" {
			t.Errorf("Markers[1] = %q, want %q", cfg.Markers[1], "go.mod")
		}

		if len(cfg.Excludes) != 2 {
			t.Errorf("Excludes length = %d, want 2", len(cfg.Excludes))
		}
		if cfg.Excludes[1] != "vendor" {
			t.Errorf("Excludes[1] = %q, want %q", cfg.Excludes[1], "vendor")
		}
	})

	t.Run("non-zero MaxDepth overrides", func(t *testing.T) {
		cfg := &Config{
			MaxDepth: 5,
		}

		flags := struct {
			MaxDepth int
		}{
			MaxDepth: 10,
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		// MaxDepth should be overridden
		if cfg.MaxDepth != 10 {
			t.Errorf("MaxDepth = %d, want 10", cfg.MaxDepth)
		}
	})

	t.Run("pointer vs value struct", func(t *testing.T) {
		cfg := &Config{
			SearchPaths: []string{"/original"},
		}

		// Test with pointer
		flagsPtr := &struct {
			Path []string
		}{
			Path: []string{"/new1"},
		}

		err := cfg.MergeFlags(flagsPtr)
		if err != nil {
			t.Fatalf("MergeFlags() with pointer error = %v, want nil", err)
		}
		if len(cfg.SearchPaths) != 2 {
			t.Error("MergeFlags should work with pointer struct")
		}

		// Test with value
		flagsVal := struct {
			Path []string
		}{
			Path: []string{"/new2"},
		}

		err = cfg.MergeFlags(flagsVal)
		if err != nil {
			t.Fatalf("MergeFlags() with value error = %v, want nil", err)
		}
		if len(cfg.SearchPaths) != 3 {
			t.Error("MergeFlags should work with value struct")
		}
	})

	t.Run("NoNested flag sets Nested to false", func(t *testing.T) {
		cfg := &Config{
			Nested: true,
		}

		flags := struct {
			NoNested bool
		}{
			NoNested: true,
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		if cfg.Nested {
			t.Error("Nested should be false when NoNested flag is true")
		}
	})

	t.Run("NoNested flag false doesn't change Nested", func(t *testing.T) {
		cfg := &Config{
			Nested: true,
		}

		flags := struct {
			NoNested bool
		}{
			NoNested: false,
		}

		err := cfg.MergeFlags(flags)
		if err != nil {
			t.Fatalf("MergeFlags() error = %v, want nil", err)
		}

		if !cfg.Nested {
			t.Error("Nested should remain true when NoNested flag is false")
		}
	})
}

func TestMarkerConfigFormats(t *testing.T) {
	t.Run("new format with marker and icon", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    icon: ""
  - marker: go.mod
    icon: "󰟓"
  - marker: package.json
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Check that markers merge with defaults (13 defaults, these 3 overlap)
		if len(cfg.Markers) != 13 {
			t.Errorf("Markers length = %d, want 13 (merged with defaults)", len(cfg.Markers))
		}

		// Check icons are populated from new format (overriding defaults)
		if cfg.Icons[".git"] != "" {
			t.Errorf("Icons[.git] = %q, want %q", cfg.Icons[".git"], "")
		}
		if cfg.Icons["go.mod"] != "󰟓" {
			t.Errorf("Icons[go.mod] = %q, want %q", cfg.Icons["go.mod"], "󰟓")
		}
		// package.json has no icon in new format, but should get default icon
		if cfg.Icons["package.json"] != "\U000f0399" {
			t.Errorf("Icons[package.json] = %q, want default icon %q", cfg.Icons["package.json"], "\U000f0399")
		}
	})

	t.Run("old format with separate markers and icons", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - .git
  - go.mod
  - package.json
icons:
  .git: ""
  go.mod: "󰟓"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Check that markers merge with defaults
		if len(cfg.Markers) != 13 {
			t.Errorf("Markers length = %d, want 13 (merged with defaults)", len(cfg.Markers))
		}

		// Check icons from old format
		if cfg.Icons[".git"] != "" {
			t.Errorf("Icons[.git] = %q, want %q", cfg.Icons[".git"], "")
		}
		if cfg.Icons["go.mod"] != "󰟓" {
			t.Errorf("Icons[go.mod] = %q, want %q", cfg.Icons["go.mod"], "󰟓")
		}
	})

	t.Run("mixed format - new format wins", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    icon: "NEW_ICON"
  - marker: go.mod
icons:
  .git: "OLD_ICON"
  go.mod: "OLD_GO_ICON"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// New format should win for .git
		if cfg.Icons[".git"] != "NEW_ICON" {
			t.Errorf("Icons[.git] = %q, want %q (new format should win)", cfg.Icons[".git"], "NEW_ICON")
		}

		// Old format should be used for go.mod (no icon in new format)
		if cfg.Icons["go.mod"] != "OLD_GO_ICON" {
			t.Errorf("Icons[go.mod] = %q, want %q", cfg.Icons["go.mod"], "OLD_GO_ICON")
		}
	})

	t.Run("mixed list with strings and objects", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - .git
  - marker: go.mod
    icon: "󰟓"
  - package.json
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Check that markers merge with defaults
		if len(cfg.Markers) != 13 {
			t.Errorf("Markers length = %d, want 13 (merged with defaults)", len(cfg.Markers))
		}

		// go.mod should have the custom icon from config
		if cfg.Icons["go.mod"] != "󰟓" {
			t.Errorf("Icons[go.mod] = %q, want %q", cfg.Icons["go.mod"], "󰟓")
		}
	})

	t.Run("invalid marker config without marker field", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - icon: ""
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() should return error for marker config without 'marker' field")
		}
	})

	t.Run("default icons preserved for markers not in config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// User config only specifies one marker with a custom icon
		yamlContent := `markers:
  - marker: .git
    icon: "custom-git"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Custom icon should be used for .git
		if cfg.Icons[".git"] != "custom-git" {
			t.Errorf("Icons[.git] = %q, want %q", cfg.Icons[".git"], "custom-git")
		}

		// Default icons should be preserved for other markers
		if cfg.Icons["go.mod"] != "\U000f07d3" {
			t.Errorf("Icons[go.mod] = %q, want default icon %q", cfg.Icons["go.mod"], "\U000f07d3")
		}
		if cfg.Icons["package.json"] != "\U000f0399" {
			t.Errorf("Icons[package.json] = %q, want default icon %q", cfg.Icons["package.json"], "\U000f0399")
		}
	})

	t.Run("old format icons merge with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// User config uses old format with custom icon for one marker
		yamlContent := `markers:
  - .git
icons:
  .git: "custom-git"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Custom icon should be used for .git
		if cfg.Icons[".git"] != "custom-git" {
			t.Errorf("Icons[.git] = %q, want %q", cfg.Icons[".git"], "custom-git")
		}

		// Default icons should be preserved for other markers
		if cfg.Icons["go.mod"] != "\U000f07d3" {
			t.Errorf("Icons[go.mod] = %q, want default icon %q", cfg.Icons["go.mod"], "\U000f07d3")
		}
	})
}

func TestLoadWithVerbose(t *testing.T) {
	t.Run("deprecation warning for old icons field", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - .git
icons:
  .git: "old-icon"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := LoadWithVerbose(configPath, true)
		if err != nil {
			t.Fatalf("LoadWithVerbose() error = %v", err)
		}

		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stderr = oldStderr

		output := buf.String()
		if !bytes.Contains([]byte(output), []byte("deprecated")) {
			t.Errorf("Expected deprecation warning, got: %q", output)
		}
	})

	t.Run("conflict warning when both formats have icons", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    icon: "new-icon"
icons:
  .git: "old-icon"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := LoadWithVerbose(configPath, true)
		if err != nil {
			t.Fatalf("LoadWithVerbose() error = %v", err)
		}

		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stderr = oldStderr

		output := buf.String()
		if !bytes.Contains([]byte(output), []byte("precedence")) {
			t.Errorf("Expected conflict warning about precedence, got: %q", output)
		}
	})

	t.Run("no warning when verbose is false", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - .git
icons:
  .git: "old-icon"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := LoadWithVerbose(configPath, false)
		if err != nil {
			t.Fatalf("LoadWithVerbose() error = %v", err)
		}

		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stderr = oldStderr

		output := buf.String()
		if len(output) > 0 {
			t.Errorf("Expected no output when verbose is false, got: %q", output)
		}
	})

	t.Run("no warning when using new format only", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    icon: "new-icon"
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_, err := LoadWithVerbose(configPath, true)
		if err != nil {
			t.Fatalf("LoadWithVerbose() error = %v", err)
		}

		_ = w.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		os.Stderr = oldStderr

		output := buf.String()
		if len(output) > 0 {
			t.Errorf("Expected no warning when using new format only, got: %q", output)
		}
	})

	t.Run("custom priority in new format", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    priority: 100
  - marker: go.mod
    priority: 5
  - marker: custom-marker
    priority: 20
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Custom priority should override default for .git
		if cfg.Priorities[".git"] != 100 {
			t.Errorf("Priorities[.git] = %d, want 100", cfg.Priorities[".git"])
		}

		// Custom priority should override default for go.mod
		if cfg.Priorities["go.mod"] != 5 {
			t.Errorf("Priorities[go.mod] = %d, want 5", cfg.Priorities["go.mod"])
		}

		// Custom marker should have custom priority
		if cfg.Priorities["custom-marker"] != 20 {
			t.Errorf("Priorities[custom-marker] = %d, want 20", cfg.Priorities["custom-marker"])
		}

		// Default priorities should be preserved for markers not in config
		if cfg.Priorities["package.json"] != 10 {
			t.Errorf("Priorities[package.json] = %d, want 10 (default)", cfg.Priorities["package.json"])
		}
	})

	t.Run("priority without icon", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		yamlContent := `markers:
  - marker: .git
    priority: 50
`
		if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Priority should be set
		if cfg.Priorities[".git"] != 50 {
			t.Errorf("Priorities[.git] = %d, want 50", cfg.Priorities[".git"])
		}

		// Default icon should be preserved
		if cfg.Icons[".git"] != "\ue65d" {
			t.Errorf("Icons[.git] = %q, want default icon", cfg.Icons[".git"])
		}
	})
}
