package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()

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

	// Check markers
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
	}
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

	// Check icons map
	if len(cfg.Icons) == 0 {
		t.Error("defaults() should have icons")
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
}
