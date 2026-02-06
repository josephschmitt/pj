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
	deepProjectPath := filepath.Join(tmpDir, "deep", "l1", "l2", "l3", "l4")
	for _, p := range projects {
		if strings.HasPrefix(p.Path, deepProjectPath) {
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
	// Create a temporary directory to use as our mock HOME
	tmpHome := t.TempDir()

	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	// Create a project directory inside our mock home
	projectDir := filepath.Join(tmpHome, "projects")
	createProject(t, projectDir, "homeproject", ".git/")

	// Use tilde path to reference the projects directory
	tildePath := "~/projects"

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

	// Verify the project path was correctly expanded
	expectedPath := filepath.Join(tmpHome, "projects", "homeproject")
	if len(projects) == 1 && projects[0].Path != expectedPath {
		t.Errorf("Project path = %v, want %v", projects[0].Path, expectedPath)
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

func TestDiscoverIDEMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create projects with IDE markers only (directories)
	createProject(t, tmpDir, "vscode-project", ".vscode/")
	createProject(t, tmpDir, "idea-project", ".idea/")
	createProject(t, tmpDir, "fleet-project", ".fleet/")
	createProject(t, tmpDir, "zed-project", ".zed/")
	// Eclipse uses a file marker
	createProject(t, tmpDir, "eclipse-project", ".project")

	// Create project with IDE marker AND language marker (language should win)
	createProject(t, tmpDir, "mixed-project", ".vscode/", "go.mod")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".vscode", ".idea", ".fleet", ".project", ".zed", "go.mod"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 6 {
		t.Fatalf("Discover() found %d projects, want 6", len(projects))
	}

	// Verify IDE-only projects use their IDE marker
	for _, p := range projects {
		switch filepath.Base(p.Path) {
		case "vscode-project":
			if p.Marker != ".vscode" {
				t.Errorf("vscode-project marker = %q, want .vscode", p.Marker)
			}
		case "idea-project":
			if p.Marker != ".idea" {
				t.Errorf("idea-project marker = %q, want .idea", p.Marker)
			}
		case "fleet-project":
			if p.Marker != ".fleet" {
				t.Errorf("fleet-project marker = %q, want .fleet", p.Marker)
			}
		case "eclipse-project":
			if p.Marker != ".project" {
				t.Errorf("eclipse-project marker = %q, want .project", p.Marker)
			}
		case "zed-project":
			if p.Marker != ".zed" {
				t.Errorf("zed-project marker = %q, want .zed", p.Marker)
			}
		case "mixed-project":
			// go.mod (priority 10) should win over .vscode (priority 1)
			if p.Marker != "go.mod" {
				t.Errorf("mixed-project marker = %q, want go.mod (higher priority)", p.Marker)
			}
		}
	}
}

func TestDiscoverDockerfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with only Dockerfile
	createProject(t, tmpDir, "docker-only", "Dockerfile")

	// Create project with Dockerfile and generic marker (Dockerfile should win - priority 7 > 1)
	createProject(t, tmpDir, "docker-with-git", "Dockerfile", ".git/")

	// Create project with Dockerfile and language marker (language should win - priority 10 > 7)
	createProject(t, tmpDir, "docker-with-gomod", "Dockerfile", "go.mod")

	// Create project with Dockerfile and IDE marker (Dockerfile should win - priority 7 > 5)
	createProject(t, tmpDir, "docker-with-vscode", "Dockerfile", ".vscode/")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{"Dockerfile", ".git", "go.mod", ".vscode"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 4 {
		t.Fatalf("Discover() found %d projects, want 4", len(projects))
	}

	for _, p := range projects {
		switch filepath.Base(p.Path) {
		case "docker-only":
			if p.Marker != "Dockerfile" {
				t.Errorf("docker-only marker = %q, want Dockerfile", p.Marker)
			}
		case "docker-with-git":
			if p.Marker != "Dockerfile" {
				t.Errorf("docker-with-git marker = %q, want Dockerfile (priority 7 > 1)", p.Marker)
			}
		case "docker-with-gomod":
			if p.Marker != "go.mod" {
				t.Errorf("docker-with-gomod marker = %q, want go.mod (priority 10 > 7)", p.Marker)
			}
		case "docker-with-vscode":
			if p.Marker != "Dockerfile" {
				t.Errorf("docker-with-vscode marker = %q, want Dockerfile (priority 7 > 5)", p.Marker)
			}
		}
	}
}

func TestDiscoverCustomPriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with .git and go.mod
	// Normally go.mod (10) wins over .git (1)
	// But with custom priority, we can make .git win
	createProject(t, tmpDir, "custom-priority", ".git/", "go.mod")

	// Test with default priorities - go.mod should win
	t.Run("default priorities", func(t *testing.T) {
		cfg := &config.Config{
			SearchPaths: []string{tmpDir},
			Markers:     []string{".git", "go.mod"},
			MaxDepth:    3,
			Excludes:    []string{},
			Priorities: map[string]int{
				".git":   1,
				"go.mod": 10,
			},
		}

		d := New(cfg, false)
		projects, err := d.Discover()
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		if len(projects) != 1 {
			t.Fatalf("Discover() found %d projects, want 1", len(projects))
		}

		if projects[0].Marker != "go.mod" {
			t.Errorf("marker = %q, want go.mod (higher default priority)", projects[0].Marker)
		}
	})

	// Test with custom priorities - .git should win with higher priority
	t.Run("custom priority overrides default", func(t *testing.T) {
		cfg := &config.Config{
			SearchPaths: []string{tmpDir},
			Markers:     []string{".git", "go.mod"},
			MaxDepth:    3,
			Excludes:    []string{},
			Priorities: map[string]int{
				".git":   100, // Override to higher priority
				"go.mod": 10,
			},
		}

		d := New(cfg, false)
		projects, err := d.Discover()
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		if len(projects) != 1 {
			t.Fatalf("Discover() found %d projects, want 1", len(projects))
		}

		if projects[0].Marker != ".git" {
			t.Errorf("marker = %q, want .git (custom higher priority)", projects[0].Marker)
		}

		if projects[0].Priority != 100 {
			t.Errorf("priority = %d, want 100", projects[0].Priority)
		}
	})
}

func TestDiscoverNested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure:
	// tmpDir/
	//   outer/
	//     .git/
	//     apps/
	//       app1/
	//         package.json
	//     packages/
	//       pkg1/
	//         go.mod

	outerDir := createProject(t, tmpDir, "outer", ".git/")

	appsDir := filepath.Join(outerDir, "apps")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}
	createProject(t, appsDir, "app1", "package.json")

	packagesDir := filepath.Join(outerDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	createProject(t, packagesDir, "pkg1", "go.mod")

	tests := []struct {
		name             string
		nested           bool
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "nested enabled finds inner projects",
			nested:           true,
			expectedCount:    3,
			shouldContain:    []string{"outer", "app1", "pkg1"},
			shouldNotContain: []string{},
		},
		{
			name:             "nested disabled skips inner projects",
			nested:           false,
			expectedCount:    1,
			shouldContain:    []string{"outer"},
			shouldNotContain: []string{"app1", "pkg1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				SearchPaths: []string{tmpDir},
				Markers:     []string{".git", "package.json", "go.mod"},
				MaxDepth:    5,
				Excludes:    []string{},
				Nested:      tt.nested,
			}

			d := New(cfg, false)
			projects, err := d.Discover()
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			if len(projects) != tt.expectedCount {
				t.Errorf("Discover() found %d projects, want %d", len(projects), tt.expectedCount)
				for _, p := range projects {
					t.Logf("  Found: %s", p.Path)
				}
			}

			found := make(map[string]bool)
			for _, p := range projects {
				found[filepath.Base(p.Path)] = true
			}

			for _, name := range tt.shouldContain {
				if !found[name] {
					t.Errorf("%s not found in results", name)
				}
			}

			for _, name := range tt.shouldNotContain {
				if found[name] {
					t.Errorf("%s found in results (should not be)", name)
				}
			}
		})
	}
}

func TestDiscoverNestedWithMaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	// tmpDir/outer/.git/
	// tmpDir/outer/level1/level2/level3/inner/go.mod
	outerDir := createProject(t, tmpDir, "outer", ".git/")
	deepPath := filepath.Join(outerDir, "level1", "level2", "level3")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatal(err)
	}
	createProject(t, deepPath, "inner", "go.mod")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "go.mod"},
		MaxDepth:    3, // Should find outer but not inner
		Excludes:    []string{},
		Nested:      true,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	// Should only find outer (inner is beyond max depth)
	if len(projects) != 1 {
		t.Errorf("Discover() with nested+maxdepth found %d projects, want 1", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	if filepath.Base(projects[0].Path) != "outer" {
		t.Errorf("Expected to find 'outer', got %s", filepath.Base(projects[0].Path))
	}
}

func TestDiscoverNestedWithExcludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create structure with excluded directory
	// tmpDir/outer/.git/
	// tmpDir/outer/node_modules/inner/go.mod (should be excluded)
	// tmpDir/outer/subproject/go.mod (should be found)
	outerDir := createProject(t, tmpDir, "outer", ".git/")

	nodeModules := filepath.Join(outerDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0755); err != nil {
		t.Fatal(err)
	}
	createProject(t, nodeModules, "inner", "go.mod")
	createProject(t, outerDir, "subproject", "go.mod")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "go.mod"},
		MaxDepth:    5,
		Excludes:    []string{"node_modules"},
		Nested:      true,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Discover() found %d projects, want 2 (outer and subproject)", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	for _, p := range projects {
		if strings.Contains(p.Path, "node_modules") {
			t.Error("Found project in node_modules (should be excluded)")
		}
	}
}

func TestDiscoverNestedWithGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	outerDir := createProject(t, tmpDir, "outer", ".git/")
	createProject(t, outerDir, "ignored-inner", "go.mod")
	createProject(t, outerDir, "visible-inner", "go.mod")

	// Create .gitignore in outer that ignores ignored-inner
	gitignorePath := filepath.Join(outerDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-inner\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "go.mod"},
		MaxDepth:    5,
		Excludes:    []string{},
		Nested:      true,
		NoIgnore:    false,
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Discover() found %d projects, want 2 (outer and visible-inner)", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s", p.Path)
		}
	}

	for _, p := range projects {
		if filepath.Base(p.Path) == "ignored-inner" {
			t.Error("Found ignored-inner (should be ignored by .gitignore)")
		}
	}
}

// Test pattern marker support
func TestDiscoverPatternMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .NET-style project with .csproj file
	dotnetDir := filepath.Join(tmpDir, "dotnet-app")
	if err := os.MkdirAll(dotnetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dotnetDir, "MyApp.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a standard Go project
	goDir := createProject(t, tmpDir, "go-app", "go.mod")

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{"go.mod", "*.csproj"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Discover() found %d projects, want 2", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s (marker: %s)", p.Path, p.Marker)
		}
	}

	foundDotnet := false
	foundGo := false
	for _, p := range projects {
		if filepath.Base(p.Path) == "dotnet-app" {
			foundDotnet = true
			if p.Marker != "MyApp.csproj" {
				t.Errorf("Expected marker 'MyApp.csproj', got '%s'", p.Marker)
			}
		}
		if filepath.Base(p.Path) == "go-app" {
			foundGo = true
		}
	}
	if !foundDotnet {
		t.Error("dotnet-app not found")
	}
	if !foundGo {
		t.Error("go-app not found (should still work with exact markers)")
	}
	_ = goDir // silence unused variable warning
}

func TestDiscoverPatternMarkerPriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project with both a .csproj and .sln file
	// .sln should win if it has higher priority
	projDir := filepath.Join(tmpDir, "multi-marker")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "App.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "Solution.sln"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{"*.csproj", "*.sln"},
		MaxDepth:    3,
		Excludes:    []string{},
		Priorities: map[string]int{
			"*.csproj": 10,
			"*.sln":    15, // Higher priority
		},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Discover() found %d projects, want 1", len(projects))
	}

	// Should find the .sln file since it has higher priority
	if projects[0].Marker != "Solution.sln" {
		t.Errorf("Expected marker 'Solution.sln' (higher priority), got '%s'", projects[0].Marker)
	}
	if projects[0].Priority != 15 {
		t.Errorf("Expected priority 15, got %d", projects[0].Priority)
	}
}

func TestDiscoverMixedExactAndPatternMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create projects with different marker types
	createProject(t, tmpDir, "git-only", ".git/")

	dotnetDir := filepath.Join(tmpDir, "dotnet-only")
	if err := os.MkdirAll(dotnetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dotnetDir, "App.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Project with both exact and pattern marker - exact should be checked first
	bothDir := filepath.Join(tmpDir, "has-both")
	if err := os.MkdirAll(filepath.Join(bothDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bothDir, "App.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{".git", "*.csproj"},
		MaxDepth:    3,
		Excludes:    []string{},
		Priorities: map[string]int{
			".git":     1,
			"*.csproj": 10, // Higher priority than .git
		},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Discover() found %d projects, want 3", len(projects))
		for _, p := range projects {
			t.Logf("  Found: %s (marker: %s)", p.Path, p.Marker)
		}
	}

	for _, p := range projects {
		if filepath.Base(p.Path) == "has-both" {
			// Pattern marker has higher priority, so .csproj should win
			if p.Marker != "App.csproj" {
				t.Errorf("Expected pattern marker 'App.csproj' (priority 10) over '.git' (priority 1), got '%s'", p.Marker)
			}
		}
	}
}

func TestDiscoverMultiplePatternMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory with multiple .csproj files
	projDir := filepath.Join(tmpDir, "multi-csproj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create multiple csproj files - first alphabetically should win
	if err := os.WriteFile(filepath.Join(projDir, "Alpha.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "Beta.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "Zeta.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{"*.csproj"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Discover() found %d projects, want 1 (directory should only appear once)", len(projects))
	}

	// First match (alphabetically from ReadDir) wins
	if projects[0].Marker != "Alpha.csproj" {
		t.Errorf("Expected first alphabetical match 'Alpha.csproj', got '%s'", projects[0].Marker)
	}
}

func TestIsPatternMarker(t *testing.T) {
	tests := []struct {
		marker   string
		expected bool
	}{
		{".git", false},
		{"go.mod", false},
		{"package.json", false},
		{"*.csproj", true},
		{"*.sln", true},
		{"test_*.yml", true},
		{"file?.txt", true},
		{"[abc].txt", true},
		{"normal-file", false},
	}

	for _, tt := range tests {
		t.Run(tt.marker, func(t *testing.T) {
			got := config.IsPatternMarker(tt.marker)
			if got != tt.expected {
				t.Errorf("IsPatternMarker(%q) = %v, want %v", tt.marker, got, tt.expected)
			}
		})
	}
}

func TestDiscoverPatternMarkerSkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory that matches the pattern (should be skipped)
	projDir := filepath.Join(tmpDir, "test-proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a directory named "Something.csproj" (unusual but possible)
	if err := os.MkdirAll(filepath.Join(projDir, "Dir.csproj"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a file that matches
	if err := os.WriteFile(filepath.Join(projDir, "App.csproj"), []byte("<Project></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		SearchPaths: []string{tmpDir},
		Markers:     []string{"*.csproj"},
		MaxDepth:    3,
		Excludes:    []string{},
	}

	d := New(cfg, false)
	projects, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Discover() found %d projects, want 1", len(projects))
	}

	// Should find the file, not the directory
	if projects[0].Marker != "App.csproj" {
		t.Errorf("Expected file marker 'App.csproj', got '%s'", projects[0].Marker)
	}
}
