package main

import (
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
