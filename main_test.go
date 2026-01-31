package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	// Binary path will be set in TestMain
	binaryPath string
)

// TestMain builds the binary before running tests and cleans up after
func TestMain(m *testing.M) {
	// Build the binary
	binaryName := "pj-test-binary"
	if os.Getenv("GOOS") == "windows" || filepath.Ext(os.Args[0]) == ".exe" {
		binaryName += ".exe"
	}
	binaryPath = filepath.Join(os.TempDir(), binaryName)
	build := exec.Command("go", "build", "-o", binaryPath)
	if err := build.Run(); err != nil {
		os.Stderr.WriteString("Failed to build binary: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(binaryPath)
	os.Exit(code)
}

// testEnv holds test environment directories
type testEnv struct {
	configDir string
	cacheDir  string
	t         *testing.T
}

// setupTestEnv creates a test environment with isolated config and cache
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	tmpConfig := t.TempDir()
	tmpCache := t.TempDir()

	// Create an empty config file to override defaults
	configDir := filepath.Join(tmpConfig, "pj")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	// Use old format with icons field for backward compatibility
	emptyConfig := `search_paths: []
markers:
  - .git
  - go.mod
  - package.json
  - Cargo.toml
  - pyproject.toml
  - Makefile
  - flake.nix
max_depth: 3
excludes:
  - node_modules
  - .terraform
  - vendor
  - .git
  - target
  - dist
  - build
cache_ttl: 300
icons:
  .git: "\ue65d"
  go.mod: "\U000f07d3"
  package.json: "\U000f0399"
  Cargo.toml: "\ue68b"
  pyproject.toml: "\ue606"
  Makefile: "\ue673"
  flake.nix: "\ue843"
`
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0644); err != nil {
		t.Fatal(err)
	}

	return &testEnv{
		configDir: tmpConfig,
		cacheDir:  tmpCache,
		t:         t,
	}
}

// runPJ runs the pj binary with arguments in the test environment
func (env *testEnv) runPJ(args ...string) (string, string, error) {
	env.t.Helper()
	cmd := exec.Command(binaryPath, args...)

	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+env.configDir,
		"XDG_CACHE_HOME="+env.cacheDir,
	)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runPJWithStdin runs the pj binary with stdin input in the test environment
func (env *testEnv) runPJWithStdin(stdin string, args ...string) (string, string, error) {
	env.t.Helper()
	cmd := exec.Command(binaryPath, args...)

	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+env.configDir,
		"XDG_CACHE_HOME="+env.cacheDir,
	)

	// Set up stdin
	cmd.Stdin = strings.NewReader(stdin)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Helper to run the pj binary with arguments (for simple tests)
func runPJ(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	env := setupTestEnv(t)
	return env.runPJ(args...)
}

// Helper to create a test project structure
func createTestProject(t *testing.T, base, name string, markers ...string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, marker := range markers {
		path := filepath.Join(dir, marker)
		if strings.HasSuffix(marker, "/") {
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(path, []byte{}, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return dir
}

func TestCLI_Version(t *testing.T) {
	stdout, _, err := runPJ(t, "--version")
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}

	if !strings.Contains(stdout, "pj version") {
		t.Errorf("--version output = %q, should contain 'pj version'", stdout)
	}
}

func TestCLI_Discovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test projects
	createTestProject(t, tmpDir, "project1", ".git/")
	createTestProject(t, tmpDir, "project2", "go.mod")
	createTestProject(t, tmpDir, "project3", "package.json")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	// Verify all projects are found
	if !strings.Contains(stdout, "project1") {
		t.Error("Output should contain project1")
	}
	if !strings.Contains(stdout, "project2") {
		t.Error("Output should contain project2")
	}
	if !strings.Contains(stdout, "project3") {
		t.Error("Output should contain project3")
	}

	// Verify output contains full paths
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines of output, got %d", len(lines))
	}

	for _, line := range lines {
		if !strings.HasPrefix(line, tmpDir) {
			t.Errorf("Output line %q should start with tmpDir %q", line, tmpDir)
		}
	}
}

func TestCLI_CacheFlow(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	// Create test project
	createTestProject(t, tmpDir, "cached-project", ".git/")

	// First run with --no-cache to force discovery
	stdout1, stderr1, err := env.runPJ("-p", tmpDir, "--no-cache", "-v")
	if err != nil {
		t.Fatalf("First run failed: %v\nStderr: %s", err, stderr1)
	}

	if !strings.Contains(stdout1, "cached-project") {
		t.Error("First run should find cached-project")
	}

	// Wait a moment to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Second run without --no-cache should hit cache
	stdout2, stderr2, err := env.runPJ("-p", tmpDir, "-v")
	if err != nil {
		t.Fatalf("Second run failed: %v\nStderr: %s", err, stderr2)
	}

	if !strings.Contains(stderr2, "Using cached results") {
		t.Errorf("Second run should use cache\nStderr: %s", stderr2)
	}

	if !strings.Contains(stdout2, "cached-project") {
		t.Error("Second run should find cached-project from cache")
	}

	// Clear cache
	_, stderr3, err := env.runPJ("-p", tmpDir, "--clear-cache", "-v")
	if err != nil {
		t.Fatalf("Clear cache failed: %v\nStderr: %s", err, stderr3)
	}

	if !strings.Contains(stderr3, "Cache cleared") {
		t.Errorf("Clear cache should confirm clearing\nStderr: %s", stderr3)
	}

	// Third run after clear should not hit cache
	stdout4, stderr4, err := env.runPJ("-p", tmpDir, "-v")
	if err != nil {
		t.Fatalf("Third run failed: %v\nStderr: %s", err, stderr4)
	}

	if strings.Contains(stderr4, "Using cached results") {
		t.Errorf("Third run should not use cache after clearing\nStderr: %s", stderr4)
	}

	if !strings.Contains(stdout4, "cached-project") {
		t.Error("Third run should find cached-project via fresh discovery")
	}
}

func TestCLI_Icons(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project with go.mod
	createTestProject(t, tmpDir, "go-project", "go.mod")

	// Run without icons flag
	stdout1, _, err := runPJ(t, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj without icons failed: %v", err)
	}

	// Should not contain icon characters
	lines1 := strings.Split(strings.TrimSpace(stdout1), "\n")
	if len(lines1) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines1))
	}

	// Run with icons flag
	stdout2, _, err := runPJ(t, "-p", tmpDir, "--no-cache", "--icons")
	if err != nil {
		t.Fatalf("pj with icons failed: %v", err)
	}

	lines2 := strings.Split(strings.TrimSpace(stdout2), "\n")
	if len(lines2) != 1 {
		t.Fatalf("Expected 1 project with icons, got %d", len(lines2))
	}

	// With --icons, the line should start with an icon followed by a space
	// The icon is a Unicode character (multi-byte), so we can check if the line is longer
	if len(lines2[0]) <= len(lines1[0]) {
		t.Error("Output with --icons should be longer than without (should include icon)")
	}

	// The lines should both end with the project path
	if !strings.HasSuffix(lines1[0], "go-project") {
		t.Error("Output without icons should end with project name")
	}
	if !strings.HasSuffix(lines2[0], "go-project") {
		t.Error("Output with icons should end with project name")
	}
}

func TestCLI_PathFlag(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create projects in different directories
	createTestProject(t, tmpDir1, "project-a", ".git/")
	createTestProject(t, tmpDir2, "project-b", ".git/")

	// Run with multiple -p flags
	stdout, stderr, err := runPJ(t, "-p", tmpDir1, "-p", tmpDir2, "--no-cache")
	if err != nil {
		t.Fatalf("pj with multiple paths failed: %v\nStderr: %s", err, stderr)
	}

	// Should find both projects
	if !strings.Contains(stdout, "project-a") {
		t.Error("Output should contain project-a")
	}
	if !strings.Contains(stdout, "project-b") {
		t.Error("Output should contain project-b")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 projects from 2 paths, got %d", len(lines))
	}
}

func TestCLI_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested projects
	createTestProject(t, tmpDir, "level1", ".git/")
	createTestProject(t, filepath.Join(tmpDir, "level1"), "level2", ".git/")
	createTestProject(t, filepath.Join(tmpDir, "level1", "level2"), "level3", ".git/")

	// Run with max depth of 1 (should only find level1)
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "-d", "1", "--no-cache")
	if err != nil {
		t.Fatalf("pj with max depth failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("With max depth 1, expected 1 project, got %d\nOutput: %s", len(lines), stdout)
	}

	if !strings.Contains(stdout, "level1") {
		t.Error("Output should contain level1")
	}
	if strings.Contains(stdout, "level2") || strings.Contains(stdout, "level3") {
		t.Error("Output should not contain level2 or level3 with max depth 1")
	}
}

func TestCLI_Excludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create projects, one in node_modules
	createTestProject(t, tmpDir, "project1", ".git/")
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestProject(t, nodeModulesDir, "excluded-project", ".git/")

	// Run normally (node_modules is in default excludes)
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "project1") {
		t.Error("Output should contain project1")
	}
	if strings.Contains(stdout, "excluded-project") {
		t.Error("Output should not contain excluded-project from node_modules")
	}
}

func TestCLI_PrioritySorting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create projects with different marker priorities
	// go.mod has priority 10, .git has priority 1
	createTestProject(t, tmpDir, "high-priority", "go.mod")
	createTestProject(t, tmpDir, "low-priority", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(lines))
	}

	// High priority (go.mod) should come first
	if !strings.Contains(lines[0], "high-priority") {
		t.Errorf("First line should be high-priority project, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "low-priority") {
		t.Errorf("Second line should be low-priority project, got: %s", lines[1])
	}
}

func TestCLI_StdinBasic(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "stdin-project", ".git/")

	stdin := tmpDir + "\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "--no-cache")
	if err != nil {
		t.Fatalf("pj with stdin failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "stdin-project") {
		t.Errorf("Output should contain stdin-project, got: %s", stdout)
	}
}

func TestCLI_StdinMultiplePaths(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir1, "project-a", "go.mod")
	createTestProject(t, tmpDir2, "project-b", "package.json")

	stdin := tmpDir1 + "\n" + tmpDir2 + "\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "--no-cache")
	if err != nil {
		t.Fatalf("pj with stdin failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "project-a") {
		t.Error("Output should contain project-a")
	}
	if !strings.Contains(stdout, "project-b") {
		t.Error("Output should contain project-b")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(lines))
	}
}

func TestCLI_StdinWithIcons(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdin := tmpDir + "\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "--no-cache", "--icons")
	if err != nil {
		t.Fatalf("pj with stdin and icons failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "go-project") {
		t.Error("Output should contain go-project")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines[0]) <= len(tmpDir)+len("/go-project") {
		t.Error("Output with --icons should include icon character")
	}
}

func TestCLI_StdinInvalidPaths(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "valid-project", ".git/")

	stdin := tmpDir + "\n/nonexistent/path\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "--no-cache", "-v")
	if err != nil {
		t.Fatalf("pj with stdin failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "valid-project") {
		t.Error("Output should contain valid-project")
	}

	if !strings.Contains(stderr, "warning: skipping invalid path") {
		t.Error("Stderr should contain warning about invalid path")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 project (invalid path should be skipped), got %d", len(lines))
	}
}

func TestCLI_StdinInvalidPathsSilent(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "valid-project", ".git/")

	stdin := tmpDir + "\n/nonexistent/path\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "--no-cache")
	if err != nil {
		t.Fatalf("pj with stdin failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "valid-project") {
		t.Error("Output should contain valid-project")
	}

	if strings.Contains(stderr, "warning") {
		t.Error("Stderr should not contain warnings without -v flag")
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 project (invalid path should be silently skipped), got %d", len(lines))
	}
}

func TestCLI_StdinSkipsCache(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "project", ".git/")

	stdin := tmpDir + "\n"
	stdout1, stderr1, err := env.runPJWithStdin(stdin, "-v")
	if err != nil {
		t.Fatalf("First stdin run failed: %v\nStderr: %s", err, stderr1)
	}

	if !strings.Contains(stdout1, "project") {
		t.Error("First run should find project")
	}

	if strings.Contains(stderr1, "Using cached results") {
		t.Error("stdin mode should not use cache on first run")
	}

	time.Sleep(100 * time.Millisecond)

	stdout2, stderr2, err := env.runPJWithStdin(stdin, "-v")
	if err != nil {
		t.Fatalf("Second stdin run failed: %v\nStderr: %s", err, stderr2)
	}

	if !strings.Contains(stdout2, "project") {
		t.Error("Second run should find project")
	}

	if strings.Contains(stderr2, "Using cached results") {
		t.Error("stdin mode should never use cache, even on subsequent runs")
	}
}

func TestCLI_StdinEmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	env := setupTestEnv(t)

	createTestProject(t, tmpDir, "project", ".git/")

	stdin := "\n\n\n"
	stdout, stderr, err := env.runPJWithStdin(stdin, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj with empty stdin failed: %v\nStderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "project") {
		t.Error("Empty stdin should fall back to normal behavior with -p flag")
	}
}

func TestCLI_NoNested(t *testing.T) {
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

	outerDir := createTestProject(t, tmpDir, "outer", ".git/")

	appsDir := filepath.Join(outerDir, "apps")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestProject(t, appsDir, "app1", "package.json")

	packagesDir := filepath.Join(outerDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestProject(t, packagesDir, "pkg1", "go.mod")

	// Default behavior (nested discovery enabled) should find all 3 projects
	stdout1, stderr1, err := runPJ(t, "-p", tmpDir, "--no-cache")
	if err != nil {
		t.Fatalf("pj without --no-nested failed: %v\nStderr: %s", err, stderr1)
	}

	lines1 := strings.Split(strings.TrimSpace(stdout1), "\n")
	if len(lines1) != 3 {
		t.Errorf("Without --no-nested, expected 3 projects, got %d\nOutput: %s", len(lines1), stdout1)
	}
	if !strings.Contains(stdout1, "outer") {
		t.Error("Output should contain outer")
	}
	if !strings.Contains(stdout1, "app1") {
		t.Error("Output should contain app1")
	}
	if !strings.Contains(stdout1, "pkg1") {
		t.Error("Output should contain pkg1")
	}

	// With --no-nested, should only find outer
	stdout2, stderr2, err := runPJ(t, "-p", tmpDir, "--no-cache", "--no-nested")
	if err != nil {
		t.Fatalf("pj with --no-nested failed: %v\nStderr: %s", err, stderr2)
	}

	lines2 := strings.Split(strings.TrimSpace(stdout2), "\n")
	if len(lines2) != 1 {
		t.Errorf("With --no-nested, expected 1 project, got %d\nOutput: %s", len(lines2), stdout2)
	}
	if !strings.Contains(stdout2, "outer") {
		t.Error("Output should contain outer")
	}
	if strings.Contains(stdout2, "app1") {
		t.Error("Output should not contain app1 with --no-nested")
	}
	if strings.Contains(stdout2, "pkg1") {
		t.Error("Output should not contain pkg1 with --no-nested")
	}
}
func TestCLI_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test projects with different markers
	createTestProject(t, tmpDir, "go-project", "go.mod")
	createTestProject(t, tmpDir, "js-project", "package.json")
	createTestProject(t, tmpDir, "rust-project", "Cargo.toml")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json")
	if err != nil {
		t.Fatalf("pj --json failed: %v\nStderr: %s", err, stderr)
	}

	// Parse JSON output
	var result struct {
		Projects []struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Marker string `json:"marker"`
			Icon   string `json:"icon,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	// Verify we found all 3 projects
	if len(result.Projects) != 3 {
		t.Errorf("Expected 3 projects in JSON, got %d", len(result.Projects))
	}

	// Verify each project has required fields
	for i, proj := range result.Projects {
		if proj.Path == "" {
			t.Errorf("Project %d missing path field", i)
		}
		if proj.Name == "" {
			t.Errorf("Project %d missing name field", i)
		}
		if proj.Marker == "" {
			t.Errorf("Project %d missing marker field", i)
		}
		if !strings.HasPrefix(proj.Path, tmpDir) {
			t.Errorf("Project %d path should start with tmpDir", i)
		}
	}

	// Verify specific project markers
	foundMarkers := make(map[string]bool)
	for _, proj := range result.Projects {
		foundMarkers[proj.Marker] = true
	}

	expectedMarkers := []string{"go.mod", "package.json", "Cargo.toml"}
	for _, marker := range expectedMarkers {
		if !foundMarkers[marker] {
			t.Errorf("Expected to find project with marker %s", marker)
		}
	}
}

func TestCLI_JSONOutputWithIcons(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json", "--icons")
	if err != nil {
		t.Fatalf("pj --json --icons failed: %v\nStderr: %s", err, stderr)
	}

	// Parse JSON output
	var result struct {
		Projects []struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Marker string `json:"marker"`
			Icon   string `json:"icon,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result.Projects))
	}

	proj := result.Projects[0]

	// Verify icon field is populated when --icons is used
	if proj.Icon == "" {
		t.Error("Icon field should be populated when --icons flag is used")
	}

	// Verify marker is go.mod
	if proj.Marker != "go.mod" {
		t.Errorf("Expected marker to be go.mod, got %s", proj.Marker)
	}

	// Verify name is correct
	if proj.Name != "go-project" {
		t.Errorf("Expected name to be go-project, got %s", proj.Name)
	}
}

func TestCLI_JSONOutputWithoutIcons(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project
	createTestProject(t, tmpDir, "project", "package.json")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json")
	if err != nil {
		t.Fatalf("pj --json failed: %v\nStderr: %s", err, stderr)
	}

	// Parse JSON output
	var result struct {
		Projects []struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Marker string `json:"marker"`
			Icon   string `json:"icon,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result.Projects))
	}

	proj := result.Projects[0]

	// Verify icon field is empty when --icons is not used
	if proj.Icon != "" {
		t.Error("Icon field should be empty when --icons flag is not used")
	}

	// Verify other fields are populated
	if proj.Path == "" || proj.Name == "" || proj.Marker == "" {
		t.Error("Path, name, and marker fields should be populated")
	}
}

func TestCLI_JSONEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create any projects

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json")
	if err != nil {
		t.Fatalf("pj --json with no projects failed: %v\nStderr: %s", err, stderr)
	}

	// Parse JSON output
	var result struct {
		Projects []struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Marker string `json:"marker"`
			Icon   string `json:"icon,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	// Verify empty array
	if len(result.Projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(result.Projects))
	}
}

func TestCLI_JSONOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "test-project", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json", "--icons")
	if err != nil {
		t.Fatalf("pj --json failed: %v\nStderr: %s", err, stderr)
	}

	// Verify output is valid JSON
	if !json.Valid([]byte(stdout)) {
		t.Errorf("Output is not valid JSON: %s", stdout)
	}

	// Verify JSON is properly indented (should have newlines and spaces)
	if !strings.Contains(stdout, "\n") {
		t.Error("JSON output should be indented (contain newlines)")
	}

	if !strings.Contains(stdout, "  ") {
		t.Error("JSON output should be indented (contain spaces)")
	}

	// Verify root structure has "projects" key
	if !strings.Contains(stdout, `"projects"`) {
		t.Error("JSON output should contain 'projects' key")
	}
}
