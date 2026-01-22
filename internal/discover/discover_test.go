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

func TestDiscoverWithGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project structure:
	// tmpDir/
	//   .gitignore (contains "ignored-project")
	//   project1/.git/
	//   ignored-project/.git/  <- should be ignored
	//   project2/.git/

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "ignored-project", ".git/")
	createProject(t, tmpDir, "project2", ".git/")

	// Create .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-project\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    false, // Respect ignore files
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only project1 and project2
	if len(projects) != 2 {
		t.Errorf("Discover() found %d projects, want 2", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	// Verify ignored-project is not in results
	for _, p := range projects {
		if filepath.Base(p.Path) == "ignored-project" {
			t.Error("Found ignored-project in results (should be ignored by .gitignore)")
		}
	}
}

func TestDiscoverWithGitignoreDirectorySlash(t *testing.T) {
	tmpDir := t.TempDir()

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "ignored-dir", ".git/")

	// Create .gitignore with trailing slash
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-dir/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only project1
	if len(projects) != 1 {
		t.Errorf("Discover() found %d projects, want 1", len(projects))
	}

	// Verify ignored-dir is not in results
	for _, p := range projects {
		if filepath.Base(p.Path) == "ignored-dir" {
			t.Error("Found ignored-dir in results (should be ignored by .gitignore pattern with /)")
		}
	}
}

func TestDiscoverWithGitignoreWildcard(t *testing.T) {
	tmpDir := t.TempDir()

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "temp-foo", ".git/")
	createProject(t, tmpDir, "temp-bar", ".git/")

	// Create .gitignore with wildcard
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("temp-*\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only project1
	if len(projects) != 1 {
		t.Errorf("Discover() found %d projects, want 1", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	// Verify temp-* projects are not in results
	for _, p := range projects {
		basename := filepath.Base(p.Path)
		if strings.HasPrefix(basename, "temp-") {
			t.Errorf("Found %s in results (should be ignored by wildcard pattern)", basename)
		}
	}
}

func TestDiscoverWithIgnoreFile(t *testing.T) {
	tmpDir := t.TempDir()

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "ignored-by-ignore", ".git/")

	// Create .ignore file (not .gitignore)
	ignorePath := filepath.Join(tmpDir, ".ignore")
	if err := os.WriteFile(ignorePath, []byte("ignored-by-ignore\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only project1
	if len(projects) != 1 {
		t.Errorf("Discover() found %d projects, want 1", len(projects))
	}

	// Verify ignored-by-ignore is not in results
	for _, p := range projects {
		if filepath.Base(p.Path) == "ignored-by-ignore" {
			t.Error("Found ignored-by-ignore in results (should be ignored by .ignore file)")
		}
	}
}

func TestDiscoverWithNoIgnoreFlag(t *testing.T) {
	tmpDir := t.TempDir()

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "ignored-project", ".git/")

	// Create .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-project\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    true, // Disable ignore file processing
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find both projects when NoIgnore is true
	if len(projects) != 2 {
		t.Errorf("Discover() with NoIgnore=true found %d projects, want 2", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	// Verify both projects are found
	found := make(map[string]bool)
	for _, p := range projects {
		found[filepath.Base(p.Path)] = true
	}

	if !found["project1"] {
		t.Error("project1 not found")
	}
	if !found["ignored-project"] {
		t.Error("ignored-project not found (should be found when NoIgnore=true)")
	}
}

func TestDiscoverHierarchicalIgnore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   .gitignore (contains "root-ignored")
	//   project1/.git/
	//   root-ignored/.git/     <- ignored by root .gitignore
	//   subdir/
	//     .gitignore (contains "sub-ignored")
	//     project2/.git/
	//     sub-ignored/.git/    <- ignored by subdir .gitignore
	//     normal/.git/

	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "root-ignored", ".git/")

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	createProject(t, subDir, "project2", ".git/")
	createProject(t, subDir, "sub-ignored", ".git/")
	createProject(t, subDir, "normal", ".git/")

	// Create root .gitignore
	rootGitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(rootGitignore, []byte("root-ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subdir .gitignore
	subGitignore := filepath.Join(subDir, ".gitignore")
	if err := os.WriteFile(subGitignore, []byte("sub-ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{},
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find project1, project2, and normal (3 total)
	if len(projects) != 3 {
		t.Errorf("Discover() found %d projects, want 3", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	// Verify expected projects are found and ignored ones are not
	found := make(map[string]bool)
	for _, p := range projects {
		found[filepath.Base(p.Path)] = true
	}

	if !found["project1"] {
		t.Error("project1 not found")
	}
	if !found["project2"] {
		t.Error("project2 not found")
	}
	if !found["normal"] {
		t.Error("normal not found")
	}
	if found["root-ignored"] {
		t.Error("root-ignored found (should be ignored by root .gitignore)")
	}
	if found["sub-ignored"] {
		t.Error("sub-ignored found (should be ignored by subdir .gitignore)")
	}
}

func TestDiscoverIgnoreWithExcludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that both ignore files and excludes work together
	createProject(t, tmpDir, "project1", ".git/")
	createProject(t, tmpDir, "ignored-by-gitignore", ".git/")
	createProject(t, tmpDir, "node_modules", ".git/")

	// Create .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-by-gitignore\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git"},
		MaxDepth:    3,
		Excludes:    []string{"node_modules"}, // Also exclude node_modules
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should find only project1
	if len(projects) != 1 {
		t.Errorf("Discover() found %d projects, want 1", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	// Verify both ignored-by-gitignore and node_modules are excluded
	for _, p := range projects {
		basename := filepath.Base(p.Path)
		if basename == "ignored-by-gitignore" {
			t.Error("Found ignored-by-gitignore (should be ignored by .gitignore)")
		}
		if basename == "node_modules" {
			t.Error("Found node_modules (should be excluded)")
		}
	}
}
