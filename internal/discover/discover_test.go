package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/josephschmitt/pj/internal/config"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pattern  string
		expected bool
	}{
		// Exact match
		{
			name:     "exact match node_modules",
			input:    "node_modules",
			pattern:  "node_modules",
			expected: true,
		},
		{
			name:     "exact match no match",
			input:    "node_modules",
			pattern:  "vendor",
			expected: false,
		},
		// Suffix patterns
		{
			name:     "suffix match success",
			input:    "test_cache",
			pattern:  "*_cache",
			expected: true,
		},
		{
			name:     "suffix match failure",
			input:    "cache_test",
			pattern:  "*_cache",
			expected: false,
		},
		// Prefix patterns
		{
			name:     "prefix match success",
			input:    ".tmp123",
			pattern:  ".tmp*",
			expected: true,
		},
		{
			name:     "prefix match failure",
			input:    "123.tmp",
			pattern:  ".tmp*",
			expected: false,
		},
		// Glob patterns
		{
			name:     "glob match success",
			input:    "test_foo_tmp",
			pattern:  "test_*_tmp",
			expected: true,
		},
		{
			name:     "glob match failure",
			input:    "test_foo_bar",
			pattern:  "test_*_tmp",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.input, tt.pattern)
			if got != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		SearchPaths: []string{"/test"},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{"node_modules"},
	}

	d := New(cfg, true)
	if d == nil {
		t.Fatal("New() returned nil")
	}

	if d.config != cfg {
		t.Error("New() should store config reference")
	}

	if !d.verbose {
		t.Error("New() should set verbose flag")
	}
}

// Helper function to create project directories with markers
func createProject(t *testing.T, base, name string, markers ...string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, marker := range markers {
		if marker == "" {
			continue // Skip empty markers
		}
		path := filepath.Join(dir, marker)
		if marker[len(marker)-1] == '/' {
			// It's a directory marker
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
		} else {
			// It's a file marker
			if err := os.WriteFile(path, []byte{}, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return dir
}

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure:
	// tmpDir/
	//   project1/
	//     .git/
	//     go.mod         <- uses go.mod (priority 10) over .git (priority 1)
	//   project2/
	//     package.json
	//   excluded/
	//     node_modules/
	//       .git/        <- should be excluded
	//   deep/
	//     l1/l2/l3/l4/
	//       .git/        <- should be excluded by max depth

	createProject(t, tmpDir, "project1", ".git/", "go.mod")
	createProject(t, tmpDir, "project2", "package.json")

	// Create excluded directory
	excludedDir := createProject(t, tmpDir, "excluded", "")
	nodeModulesDir := filepath.Join(excludedDir, "node_modules")
	if err := os.MkdirAll(filepath.Join(nodeModulesDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create deep directory beyond max depth
	deepPath := filepath.Join(tmpDir, "deep", "l1", "l2", "l3", "l4")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(deepPath, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "go.mod", "package.json"},
		MaxDepth:    3,
		Excludes:    []string{"node_modules"},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	// Verify we found exactly 2 projects
	if len(projects) != 2 {
		t.Errorf("Discover() found %d projects, want 2", len(projects))
		for i, p := range projects {
			t.Logf("  Project %d: %s (marker: %s, priority: %d)", i, p.Path, p.Marker, p.Priority)
		}
	}

	// Verify project1 uses go.mod (higher priority than .git)
	project1Found := false
	for _, p := range projects {
		if filepath.Base(p.Path) == "project1" {
			project1Found = true
			if p.Marker != "go.mod" {
				t.Errorf("project1 marker = %q, want %q (should use highest priority marker)", p.Marker, "go.mod")
			}
			if p.Priority != 10 {
				t.Errorf("project1 priority = %d, want 10", p.Priority)
			}
		}
	}
	if !project1Found {
		t.Error("project1 not found in results")
	}

	// Verify project2 is found
	project2Found := false
	for _, p := range projects {
		if filepath.Base(p.Path) == "project2" {
			project2Found = true
			if p.Marker != "package.json" {
				t.Errorf("project2 marker = %q, want %q", p.Marker, "package.json")
			}
		}
	}
	if !project2Found {
		t.Error("project2 not found in results")
	}

	// Verify node_modules was excluded
	for _, p := range projects {
		if filepath.Base(p.Path) == "node_modules" || strings.Contains(p.Path, "node_modules") {
			t.Errorf("Found project in node_modules: %s (should be excluded)", p.Path)
		}
	}

	// Verify deep directory was excluded by max depth
	for _, p := range projects {
		if strings.Contains(p.Path, "l4") {
			t.Errorf("Found project beyond max depth: %s", p.Path)
		}
	}

	// Verify results are sorted by priority then path
	if len(projects) > 1 {
		for i := 0; i < len(projects)-1; i++ {
			if projects[i].Priority < projects[i+1].Priority {
				t.Errorf("Projects not sorted by priority: project[%d].Priority = %d < project[%d].Priority = %d",
					i, projects[i].Priority, i+1, projects[i+1].Priority)
			}
			if projects[i].Priority == projects[i+1].Priority && projects[i].Path > projects[i+1].Path {
				t.Errorf("Projects with same priority not sorted by path: %q > %q",
					projects[i].Path, projects[i+1].Path)
			}
		}
	}
}

func TestDiscoverNonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent")

	cfg := &config.Config{
		SearchPaths: []string{nonExistentPath},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	// Should return empty list, not error
	if len(projects) != 0 {
		t.Errorf("Discover() with non-existent path found %d projects, want 0", len(projects))
	}
}

func TestDiscoverTildeExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	createProject(t, tmpDir, "testproject", ".git/")

	// Use tilde path (this test assumes HOME is set)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// Create a test directory in home
	testDir := filepath.Join(home, ".pj-test-"+t.Name())
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	createProject(t, testDir, "homeproject", ".git/")

	// Calculate relative path from home
	relPath, err := filepath.Rel(home, testDir)
	if err != nil {
		t.Fatal(err)
	}
	tildePath := "~/" + relPath

	cfg := &config.Config{
		SearchPaths: []string{tildePath},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	if len(projects) != 1 {
		t.Errorf("Discover() with tilde path found %d projects, want 1", len(projects))
	}
}

func TestDiscoverDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	createProject(t, tmpDir, "project1", ".git/")

	// Use the same path twice
	cfg := &config.Config{
		SearchPaths: []string{tmpDir, tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	// Should deduplicate and return only 1 project
	if len(projects) != 1 {
		t.Errorf("Discover() with duplicate paths found %d projects, want 1 (should deduplicate)", len(projects))
	}
}

func TestDiscoverMultipleMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project with multiple markers of different priorities
	createProject(t, tmpDir, "multi-marker", ".git/", "Makefile", "go.mod")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "Makefile", "go.mod"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Discover() found %d projects, want 1", len(projects))
	}

	// Should use go.mod (priority 10) over .git (priority 1) and Makefile (priority 1)
	if projects[0].Marker != "go.mod" {
		t.Errorf("Project marker = %q, want %q (should use highest priority)", projects[0].Marker, "go.mod")
	}
	if projects[0].Priority != 10 {
		t.Errorf("Project priority = %d, want 10", projects[0].Priority)
	}
}
