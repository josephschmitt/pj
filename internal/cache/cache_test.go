package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/josephschmitt/pj/internal/config"
	"github.com/josephschmitt/pj/internal/discover"
)

func TestGetCacheDir(t *testing.T) {
	t.Run("with XDG_CACHE_HOME set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		dir := getCacheDir()
		expected := filepath.Join(tmpDir, "pj")
		if dir != expected {
			t.Errorf("getCacheDir() = %q, want %q", dir, expected)
		}
	})

	t.Run("without XDG_CACHE_HOME", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")

		dir := getCacheDir()
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".cache", "pj")
		if dir != expected {
			t.Errorf("getCacheDir() = %q, want %q", dir, expected)
		}
	})
}

func TestComputeConfigHash(t *testing.T) {
	t.Run("same config produces same hash", func(t *testing.T) {
		cfg := &config.Config{
			SearchPaths: []string{"/path1", "/path2"},
			Markers:     []string{".git", "go.mod"},
			Excludes:    []string{"node_modules"},
			MaxDepth:    3,
		}

		m1 := &Manager{config: cfg}
		m2 := &Manager{config: cfg}

		hash1 := m1.computeConfigHash()
		hash2 := m2.computeConfigHash()

		if hash1 != hash2 {
			t.Errorf("Same config produced different hashes: %q vs %q", hash1, hash2)
		}
	})

	t.Run("different configs produce different hashes", func(t *testing.T) {
		cfg1 := &config.Config{
			SearchPaths: []string{"/path1"},
			Markers:     []string{".git"},
			Excludes:    []string{"node_modules"},
			MaxDepth:    3,
		}

		cfg2 := &config.Config{
			SearchPaths: []string{"/path2"},
			Markers:     []string{".git"},
			Excludes:    []string{"node_modules"},
			MaxDepth:    3,
		}

		m1 := &Manager{config: cfg1}
		m2 := &Manager{config: cfg2}

		hash1 := m1.computeConfigHash()
		hash2 := m2.computeConfigHash()

		if hash1 == hash2 {
			t.Error("Different configs produced same hash")
		}
	})

	t.Run("order-independent hashing", func(t *testing.T) {
		cfg1 := &config.Config{
			SearchPaths: []string{"/path1", "/path2"},
			Markers:     []string{".git", "go.mod"},
			Excludes:    []string{"node_modules", "vendor"},
			MaxDepth:    3,
		}

		cfg2 := &config.Config{
			SearchPaths: []string{"/path2", "/path1"},
			Markers:     []string{"go.mod", ".git"},
			Excludes:    []string{"vendor", "node_modules"},
			MaxDepth:    3,
		}

		m1 := &Manager{config: cfg1}
		m2 := &Manager{config: cfg2}

		hash1 := m1.computeConfigHash()
		hash2 := m2.computeConfigHash()

		if hash1 != hash2 {
			t.Errorf("Same config (different order) produced different hashes: %q vs %q", hash1, hash2)
		}
	})

	t.Run("different max depth produces different hash", func(t *testing.T) {
		cfg1 := &config.Config{
			SearchPaths: []string{"/path1"},
			Markers:     []string{".git"},
			Excludes:    []string{},
			MaxDepth:    3,
		}

		cfg2 := &config.Config{
			SearchPaths: []string{"/path1"},
			Markers:     []string{".git"},
			Excludes:    []string{},
			MaxDepth:    5,
		}

		m1 := &Manager{config: cfg1}
		m2 := &Manager{config: cfg2}

		hash1 := m1.computeConfigHash()
		hash2 := m2.computeConfigHash()

		if hash1 == hash2 {
			t.Error("Different MaxDepth produced same hash")
		}
	})

	t.Run("different Nested produces different hash", func(t *testing.T) {
		cfg1 := &config.Config{
			SearchPaths: []string{"/path1"},
			Markers:     []string{".git"},
			Excludes:    []string{},
			MaxDepth:    3,
			Nested:      false,
		}

		cfg2 := &config.Config{
			SearchPaths: []string{"/path1"},
			Markers:     []string{".git"},
			Excludes:    []string{},
			MaxDepth:    3,
			Nested:      true,
		}

		m1 := &Manager{config: cfg1}
		m2 := &Manager{config: cfg2}

		hash1 := m1.computeConfigHash()
		hash2 := m2.computeConfigHash()

		if hash1 == hash2 {
			t.Error("Different Nested values should produce different hashes")
		}
	})
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	cfg := &config.Config{
		SearchPaths: []string{"/test"},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		CacheTTL:    300,
	}

	m := New(cfg, true)
	if m == nil {
		t.Fatal("New() returned nil")
	}

	if m.config != cfg {
		t.Error("New() should store config reference")
	}

	if !m.verbose {
		t.Error("New() should set verbose flag")
	}

	expectedCacheDir := filepath.Join(tmpDir, "pj")
	if m.cacheDir != expectedCacheDir {
		t.Errorf("Manager cacheDir = %q, want %q", m.cacheDir, expectedCacheDir)
	}
}

func TestGet(t *testing.T) {
	t.Run("cache not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)
		_, err := m.Get()
		if err == nil {
			t.Error("Get() should return error when cache not found")
		}
		if err.Error() != "cache not found" {
			t.Errorf("Get() error = %q, want %q", err.Error(), "cache not found")
		}
	})

	t.Run("cache expired", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    1, // 1 second TTL
		}

		m := New(cfg, false)
		cacheDir := filepath.Join(tmpDir, "pj")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create cache file
		projects := []discover.Project{
			{Path: "/test/project1", Marker: ".git", Priority: 1},
		}
		data, _ := json.Marshal(projects)
		cachePath := m.getCachePath()
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		// Set modification time to 2 seconds ago (older than TTL)
		oldTime := time.Now().Add(-2 * time.Second)
		if err := os.Chtimes(cachePath, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		_, err := m.Get()
		if err == nil {
			t.Error("Get() should return error when cache expired")
		}
		if err.Error() != "cache expired" {
			t.Errorf("Get() error = %q, want %q", err.Error(), "cache expired")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)
		cacheDir := filepath.Join(tmpDir, "pj")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Write invalid JSON
		cachePath := m.getCachePath()
		if err := os.WriteFile(cachePath, []byte("invalid json"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := m.Get()
		if err == nil {
			t.Error("Get() should return error for invalid JSON")
		}
	})

	t.Run("valid cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)
		cacheDir := filepath.Join(tmpDir, "pj")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create valid cache
		expectedProjects := []discover.Project{
			{Path: "/test/project1", Marker: ".git", Priority: 1},
			{Path: "/test/project2", Marker: "go.mod", Priority: 10},
		}
		data, _ := json.MarshalIndent(expectedProjects, "", "  ")
		cachePath := m.getCachePath()
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		projects, err := m.Get()
		if err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}

		if len(projects) != len(expectedProjects) {
			t.Errorf("Get() returned %d projects, want %d", len(projects), len(expectedProjects))
		}

		for i, p := range projects {
			if p.Path != expectedProjects[i].Path {
				t.Errorf("Project[%d].Path = %q, want %q", i, p.Path, expectedProjects[i].Path)
			}
			if p.Marker != expectedProjects[i].Marker {
				t.Errorf("Project[%d].Marker = %q, want %q", i, p.Marker, expectedProjects[i].Marker)
			}
			if p.Priority != expectedProjects[i].Priority {
				t.Errorf("Project[%d].Priority = %d, want %d", i, p.Priority, expectedProjects[i].Priority)
			}
		}
	})
}

func TestSet(t *testing.T) {
	t.Run("normal write", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)

		projects := []discover.Project{
			{Path: "/test/project1", Marker: ".git", Priority: 1},
		}

		err := m.Set(projects)
		if err != nil {
			t.Fatalf("Set() error = %v, want nil", err)
		}

		// Verify file was created
		cachePath := m.getCachePath()
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("Set() did not create cache file")
		}

		// Verify content
		data, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatal(err)
		}

		var readProjects []discover.Project
		if err := json.Unmarshal(data, &readProjects); err != nil {
			t.Fatalf("Cache file contains invalid JSON: %v", err)
		}

		if len(readProjects) != len(projects) {
			t.Errorf("Cache contains %d projects, want %d", len(readProjects), len(projects))
		}
	})

	t.Run("directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentCache := filepath.Join(tmpDir, "nonexistent", "cache")
		t.Setenv("XDG_CACHE_HOME", nonExistentCache)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)

		projects := []discover.Project{
			{Path: "/test/project1", Marker: ".git", Priority: 1},
		}

		err := m.Set(projects)
		if err != nil {
			t.Fatalf("Set() error = %v, want nil (should create directory)", err)
		}

		// Verify directory was created
		if _, err := os.Stat(m.cacheDir); os.IsNotExist(err) {
			t.Error("Set() did not create cache directory")
		}
	})
}

func TestClear(t *testing.T) {
	t.Run("clear existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)

		// Create cache directory and files
		cacheDir := filepath.Join(tmpDir, "pj")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cacheDir, "cache-test.json"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		err := m.Clear()
		if err != nil {
			t.Fatalf("Clear() error = %v, want nil", err)
		}

		// Verify directory was removed
		if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
			t.Error("Clear() did not remove cache directory")
		}
	})

	t.Run("clear non-existent", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", tmpDir)

		cfg := &config.Config{
			SearchPaths: []string{"/test"},
			Markers:     []string{".git"},
			MaxDepth:    3,
			CacheTTL:    300,
		}

		m := New(cfg, false)

		// Don't create cache directory
		err := m.Clear()
		if err != nil {
			t.Errorf("Clear() on non-existent cache error = %v, want nil", err)
		}
	})
}

func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	cfg := &config.Config{
		SearchPaths: []string{"/test"},
		Markers:     []string{".git", "go.mod"},
		MaxDepth:    3,
		CacheTTL:    300,
		Excludes:    []string{"node_modules"},
	}

	m := New(cfg, false)

	originalProjects := []discover.Project{
		{Path: "/test/project1", Marker: "go.mod", Priority: 10},
		{Path: "/test/project2", Marker: ".git", Priority: 1},
		{Path: "/test/project3", Marker: "go.mod", Priority: 10},
	}

	// Set
	err := m.Set(originalProjects)
	if err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	// Get
	retrievedProjects, err := m.Get()
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	// Verify
	if len(retrievedProjects) != len(originalProjects) {
		t.Fatalf("Round trip returned %d projects, want %d", len(retrievedProjects), len(originalProjects))
	}

	for i, p := range retrievedProjects {
		if p.Path != originalProjects[i].Path {
			t.Errorf("Project[%d].Path = %q, want %q", i, p.Path, originalProjects[i].Path)
		}
		if p.Marker != originalProjects[i].Marker {
			t.Errorf("Project[%d].Marker = %q, want %q", i, p.Marker, originalProjects[i].Marker)
		}
		if p.Priority != originalProjects[i].Priority {
			t.Errorf("Project[%d].Priority = %d, want %d", i, p.Priority, originalProjects[i].Priority)
		}
	}
}
