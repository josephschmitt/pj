package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/josephschmitt/pj/internal/discover"
	"github.com/josephschmitt/pj/internal/icons"
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
		_, _ = os.Stderr.WriteString("Failed to build binary: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	_ = os.Remove(binaryPath)
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
  - Dockerfile
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
  Dockerfile: "\ue7b0"
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

	// Default sort is priority descending
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

func TestCLI_LabelsDefault(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	// --labels without a value should default to "label"
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--labels")
	if err != nil {
		t.Fatalf("pj --labels (no value) failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "go ") {
		t.Errorf("Output should start with 'go ' label, got: %s", lines[0])
	}
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Errorf("Output should end with project name, got: %s", lines[0])
	}
}

func TestCLI_LabelsLabel(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--labels", "label")
	if err != nil {
		t.Fatalf("pj --labels label failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "go ") {
		t.Errorf("Output should start with 'go ' label, got: %s", lines[0])
	}
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Errorf("Output should end with project name, got: %s", lines[0])
	}
}

func TestCLI_LabelsDisplay(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--labels", "display")
	if err != nil {
		t.Fatalf("pj --labels display failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "Go ") {
		t.Errorf("Output should start with 'Go ' label, got: %s", lines[0])
	}
}

func TestCLI_LabelsWithIcons(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--labels", "label", "--icons")
	if err != nil {
		t.Fatalf("pj --labels --icons failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	// Should have both icon and label (label is raw, no brackets)
	if !strings.Contains(lines[0], "go ") {
		t.Errorf("Output should contain 'go' label, got: %s", lines[0])
	}
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Errorf("Output should end with project name, got: %s", lines[0])
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
			Path               string `json:"path"`
			Name               string `json:"name"`
			Marker             string `json:"marker"`
			MarkerLabel        string `json:"markerLabel"`
			MarkerDisplayLabel string `json:"markerDisplayLabel,omitempty"`
			Icon               string `json:"icon,omitempty"`
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
		if proj.MarkerLabel == "" {
			t.Errorf("Project %d missing markerLabel field", i)
		}
		if !strings.HasPrefix(proj.Path, tmpDir) {
			t.Errorf("Project %d path should start with tmpDir", i)
		}
	}

	// Verify specific project markers and their labels
	foundLabels := make(map[string]string)
	foundDisplayLabels := make(map[string]string)
	for _, proj := range result.Projects {
		foundLabels[proj.Marker] = proj.MarkerLabel
		foundDisplayLabels[proj.Marker] = proj.MarkerDisplayLabel
	}

	expectedMarkerLabels := map[string]string{
		"go.mod":       "go",
		"package.json": "nodejs",
		"Cargo.toml":   "rust",
	}
	for marker, expectedLabel := range expectedMarkerLabels {
		if foundLabels[marker] != expectedLabel {
			t.Errorf("MarkerLabel for %s = %q, want %q", marker, foundLabels[marker], expectedLabel)
		}
	}

	expectedDisplayLabels := map[string]string{
		"go.mod":       "Go",
		"package.json": "NodeJS",
		"Cargo.toml":   "Rust",
	}
	for marker, expectedDisplayLabel := range expectedDisplayLabels {
		if foundDisplayLabels[marker] != expectedDisplayLabel {
			t.Errorf("MarkerDisplayLabel for %s = %q, want %q", marker, foundDisplayLabels[marker], expectedDisplayLabel)
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
			Color  string `json:"color,omitempty"`
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

	// Verify color field is populated when --icons is used (even without --ansi)
	if proj.Color == "" {
		t.Error("Color field should be populated when --icons flag is used")
	}
	if proj.Color != "cyan" {
		t.Errorf("Color = %q, want %q (go.mod default)", proj.Color, "cyan")
	}

	// Verify marker is go.mod
	if proj.Marker != "go.mod" {
		t.Errorf("Expected marker to be go.mod, got %s", proj.Marker)
	}

	// Verify name is correct
	if proj.Name != "go-project" {
		t.Errorf("Expected name to be go-project, got %s", proj.Name)
	}

	// Verify markerLabel is always present in JSON (even raw access)
	var rawResult map[string][]map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &rawResult); err != nil {
		t.Fatalf("Failed to parse raw JSON output: %v", err)
	}
	rawProj := rawResult["projects"][0]
	if _, exists := rawProj["markerLabel"]; !exists {
		t.Error("markerLabel field should always be present in JSON output")
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

func TestCLI_AnsiIcons(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	// Run with --icons --ansi
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--icons", "--ansi")
	if err != nil {
		t.Fatalf("pj --icons --ansi failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	// Output should contain ANSI escape codes (\033[)
	if !strings.Contains(lines[0], "\033[") {
		t.Error("Output with --icons --ansi should contain ANSI escape codes")
	}

	// Output should contain the reset sequence \033[39m
	if !strings.Contains(lines[0], "\033[39m") {
		t.Error("Output with --icons --ansi should contain ANSI reset sequence")
	}

	// Should still end with the project path
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Error("Output with --icons --ansi should end with project name")
	}
}

func TestCLI_AnsiWithoutIcons(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	// Run with just --ansi (no --icons)
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--ansi")
	if err != nil {
		t.Fatalf("pj --ansi failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	// Without --icons, output should NOT contain ANSI escape codes
	if strings.Contains(lines[0], "\033[") {
		t.Error("Output with --ansi but without --icons should not contain ANSI escape codes")
	}
}

func TestCLI_AnsiColorMap(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	// Run with --icons --ansi --color-map go.mod:cyan
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--icons", "--ansi", "--color-map", "go.mod:cyan")
	if err != nil {
		t.Fatalf("pj --icons --ansi --color-map failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	// Cyan is ANSI code 36, so output should contain \033[36m
	if !strings.Contains(lines[0], "\033[36m") {
		t.Error("Output with --color-map go.mod:cyan should contain cyan ANSI code (\\033[36m)")
	}
}

func TestCLI_AnsiJSON(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json", "--icons", "--ansi")
	if err != nil {
		t.Fatalf("pj --json --icons --ansi failed: %v\nStderr: %s", err, stderr)
	}

	var result struct {
		Projects []struct {
			Path     string `json:"path"`
			Name     string `json:"name"`
			Marker   string `json:"marker"`
			Icon     string `json:"icon,omitempty"`
			AnsiIcon string `json:"ansiIcon,omitempty"`
			Color    string `json:"color,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result.Projects))
	}

	proj := result.Projects[0]

	if proj.Color == "" {
		t.Error("Color field should be populated when --icons --ansi are used")
	}
	if proj.Color != "cyan" {
		t.Errorf("Color = %q, want %q (go.mod default)", proj.Color, "cyan")
	}

	// Icon in JSON should be plain (not ANSI-wrapped)
	if strings.Contains(proj.Icon, "\033[") {
		t.Error("Icon in JSON output should not contain ANSI codes")
	}

	// AnsiIcon should contain ANSI codes when --ansi is used
	if proj.AnsiIcon == "" {
		t.Error("AnsiIcon field should be populated when --icons --ansi are used")
	}
	if !strings.Contains(proj.AnsiIcon, "\033[") {
		t.Error("AnsiIcon should contain ANSI escape codes")
	}
	if !strings.Contains(proj.AnsiIcon, proj.Icon) {
		t.Errorf("AnsiIcon %q should contain the plain icon %q", proj.AnsiIcon, proj.Icon)
	}
}

func TestCLI_AnsiJSONDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json", "--icons")
	if err != nil {
		t.Fatalf("pj --json --icons failed: %v\nStderr: %s", err, stderr)
	}

	var result struct {
		Projects []struct {
			AnsiIcon string `json:"ansiIcon,omitempty"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result.Projects))
	}

	if result.Projects[0].AnsiIcon != "" {
		t.Error("AnsiIcon should be empty when --ansi is not used")
	}
}

// setFakeHome sets the home directory env vars for all platforms.
// On Windows, os.UserHomeDir() checks USERPROFILE before HOME.
func setFakeHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

func TestShortenHome(t *testing.T) {
	sep := string(os.PathSeparator)
	homeDir := filepath.FromSlash("/Users/joe")

	tests := []struct {
		name     string
		path     string
		homeDir  string
		expected string
	}{
		{name: "path under home", path: filepath.FromSlash("/Users/joe/projects/pj"), homeDir: homeDir, expected: "~" + sep + filepath.Join("projects", "pj")},
		{name: "path is home exactly", path: filepath.FromSlash("/Users/joe"), homeDir: homeDir, expected: "~"},
		{name: "path outside home", path: filepath.FromSlash("/opt/projects/pj"), homeDir: homeDir, expected: filepath.FromSlash("/opt/projects/pj")},
		{name: "empty homeDir", path: filepath.FromSlash("/Users/joe/projects/pj"), homeDir: "", expected: filepath.FromSlash("/Users/joe/projects/pj")},
		{name: "path with similar prefix", path: filepath.FromSlash("/Users/joextra/projects"), homeDir: homeDir, expected: filepath.FromSlash("/Users/joextra/projects")},
		{name: "path is home with trailing sep", path: homeDir + sep, homeDir: homeDir, expected: "~" + sep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortenHome(tt.path, tt.homeDir)
			if result != tt.expected {
				t.Errorf("shortenHome(%q, %q) = %q, want %q", tt.path, tt.homeDir, result, tt.expected)
			}
		})
	}
}

func TestCLI_Shorten(t *testing.T) {
	fakeHome := t.TempDir()
	setFakeHome(t, fakeHome)

	createTestProject(t, fakeHome, "my-project", ".git/")

	tildePrefix := "~" + string(os.PathSeparator)

	env := setupTestEnv(t)
	stdout, stderr, err := env.runPJ("-p", fakeHome, "--no-cache", "--shorten")
	if err != nil {
		t.Fatalf("pj --shorten failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d\nOutput: %s", len(lines), stdout)
	}

	if !strings.HasPrefix(lines[0], tildePrefix) {
		t.Errorf("Output should start with %s, got: %s", tildePrefix, lines[0])
	}
	if strings.Contains(lines[0], fakeHome) {
		t.Errorf("Output should not contain absolute home path, got: %s", lines[0])
	}
}

func TestCLI_ShortenWithIcons(t *testing.T) {
	fakeHome := t.TempDir()
	setFakeHome(t, fakeHome)

	createTestProject(t, fakeHome, "go-project", "go.mod")

	tildePrefix := "~" + string(os.PathSeparator)

	env := setupTestEnv(t)
	stdout, stderr, err := env.runPJ("-p", fakeHome, "--no-cache", "--shorten", "--icons")
	if err != nil {
		t.Fatalf("pj --shorten --icons failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	if !strings.Contains(lines[0], tildePrefix) {
		t.Errorf("Output with --shorten --icons should contain %s, got: %s", tildePrefix, lines[0])
	}
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Errorf("Output should end with project name, got: %s", lines[0])
	}
}

func TestCLI_ShortenJSON(t *testing.T) {
	fakeHome := t.TempDir()
	setFakeHome(t, fakeHome)

	createTestProject(t, fakeHome, "go-project", "go.mod")

	env := setupTestEnv(t)
	stdout, stderr, err := env.runPJ("-p", fakeHome, "--no-cache", "--shorten", "--json")
	if err != nil {
		t.Fatalf("pj --shorten --json failed: %v\nStderr: %s", err, stderr)
	}

	var result struct {
		Projects []struct {
			Path        string `json:"path"`
			DisplayPath string `json:"displayPath,omitempty"`
			Name        string `json:"name"`
			Marker      string `json:"marker"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(result.Projects))
	}

	proj := result.Projects[0]

	// path should remain absolute
	if !strings.HasPrefix(proj.Path, fakeHome) {
		t.Errorf("path should be absolute, got: %s", proj.Path)
	}

	// displayPath should be shortened
	tildePrefix := "~" + string(os.PathSeparator)
	if !strings.HasPrefix(proj.DisplayPath, tildePrefix) {
		t.Errorf("displayPath should start with %s, got: %s", tildePrefix, proj.DisplayPath)
	}

	// name should still be correct
	if proj.Name != "go-project" {
		t.Errorf("name = %q, want %q", proj.Name, "go-project")
	}
}

func TestCLI_ShortenJSONDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "project", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json")
	if err != nil {
		t.Fatalf("pj --json failed: %v\nStderr: %s", err, stderr)
	}

	// displayPath should not appear in JSON when --shorten is not used
	if strings.Contains(stdout, "displayPath") {
		t.Error("JSON output should not contain displayPath when --shorten is not used")
	}
}

func TestCLI_ShortenNonHomePath(t *testing.T) {
	fakeHome := t.TempDir()
	setFakeHome(t, fakeHome)

	// Create project outside of fake home
	otherDir := t.TempDir()
	createTestProject(t, otherDir, "other-project", ".git/")

	env := setupTestEnv(t)
	stdout, stderr, err := env.runPJ("-p", otherDir, "--no-cache", "--shorten")
	if err != nil {
		t.Fatalf("pj --shorten failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	// Path outside home should not be shortened
	tildePrefix := "~" + string(os.PathSeparator)
	if strings.HasPrefix(lines[0], tildePrefix) {
		t.Errorf("Path outside home should not start with %s, got: %s", tildePrefix, lines[0])
	}
	if !strings.HasPrefix(lines[0], otherDir) {
		t.Errorf("Path outside home should remain absolute, got: %s", lines[0])
	}
}

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		values   map[string]string
		expected string
	}{
		{
			name:     "simple path",
			format:   "%p",
			values:   map[string]string{"%p": "/home/user/project"},
			expected: "/home/user/project",
		},
		{
			name:     "multiple placeholders",
			format:   "%i %p",
			values:   map[string]string{"%i": "X", "%p": "/path"},
			expected: "X /path",
		},
		{
			name:     "escape percent",
			format:   "%%p: %p",
			values:   map[string]string{"%p": "/actual/path"},
			expected: "%p: /actual/path",
		},
		{
			name:     "all placeholders",
			format:   "%i [%l] %L %n %m %c %p %P",
			values:   map[string]string{"%i": "X", "%l": "go", "%L": "Go", "%n": "myproj", "%m": "go.mod", "%c": "cyan", "%p": "~/dev/myproj", "%P": "/home/user/dev/myproj"},
			expected: "X [go] Go myproj go.mod cyan ~/dev/myproj /home/user/dev/myproj",
		},
		{
			name:     "P before p ordering",
			format:   "%P %p",
			values:   map[string]string{"%P": "/full/path", "%p": "~/short"},
			expected: "/full/path ~/short",
		},
		{
			name:     "L before l ordering",
			format:   "%L %l",
			values:   map[string]string{"%L": "Go", "%l": "go"},
			expected: "Go go",
		},
		{
			name:     "no placeholders",
			format:   "hello world",
			values:   map[string]string{"%p": "/path"},
			expected: "hello world",
		},
		{
			name:     "empty values",
			format:   "%i%p",
			values:   map[string]string{"%i": "", "%p": "/path"},
			expected: "/path",
		},
		{
			name:     "double percent at end",
			format:   "100%%",
			values:   map[string]string{},
			expected: "100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOutput(tt.format, tt.values)
			if got != tt.expected {
				t.Errorf("formatOutput(%q) = %q, want %q", tt.format, got, tt.expected)
			}
		})
	}
}

func TestCLI_FormatPath(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%p")
	if err != nil {
		t.Fatalf("pj --format failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(lines))
	}

	expected := filepath.Join(tmpDir, "go-project")
	if lines[0] != expected {
		t.Errorf("Output = %q, want %q", lines[0], expected)
	}
}

func TestCLI_FormatFullPath(t *testing.T) {
	fakeHome := t.TempDir()
	setFakeHome(t, fakeHome)
	createTestProject(t, fakeHome, "project", "go.mod")

	env := setupTestEnv(t)

	// %P should always show full path even with --shorten
	stdout, stderr, err := env.runPJ("-p", fakeHome, "--no-cache", "--shorten", "--format", "%P")
	if err != nil {
		t.Fatalf("pj --format %%P failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := filepath.Join(fakeHome, "project")
	if lines[0] != expected {
		t.Errorf("%%P output = %q, want %q (full path)", lines[0], expected)
	}

	// %p should be shortened with --shorten
	stdout2, _, err := env.runPJ("-p", fakeHome, "--no-cache", "--shorten", "--format", "%p")
	if err != nil {
		t.Fatalf("pj --format %%p --shorten failed: %v", err)
	}

	lines2 := strings.Split(strings.TrimSpace(stdout2), "\n")
	if !strings.HasPrefix(lines2[0], "~") {
		t.Errorf("%%p with --shorten should start with ~, got: %s", lines2[0])
	}
}

func TestCLI_FormatName(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "my-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%n %m")
	if err != nil {
		t.Fatalf("pj --format failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if lines[0] != "my-project go.mod" {
		t.Errorf("Output = %q, want %q", lines[0], "my-project go.mod")
	}
}

func TestCLI_FormatIcon(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	// With --icons, %i should have content
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--icons", "--format", "%i %p")
	if err != nil {
		t.Fatalf("pj --format --icons failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if !strings.HasSuffix(lines[0], "go-project") {
		t.Errorf("Output should end with project name, got: %s", lines[0])
	}
	// Icon adds content before the space, so line should be longer than just the path
	plainPath := filepath.Join(tmpDir, "go-project")
	if len(lines[0]) <= len(plainPath)+1 {
		t.Errorf("Output with %%i --icons should include icon, got: %s", lines[0])
	}
}

func TestCLI_FormatIconWithoutFlag(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	// Without --icons, %i should be empty
	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%i%p")
	if err != nil {
		t.Fatalf("pj --format without --icons failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := filepath.Join(tmpDir, "go-project")
	if lines[0] != expected {
		t.Errorf("%%i without --icons should be empty, got: %q, want %q", lines[0], expected)
	}
}

func TestCLI_FormatLabel(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "[%l] %p")
	if err != nil {
		t.Fatalf("pj --format label failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if !strings.HasPrefix(lines[0], "[go] ") {
		t.Errorf("Output should start with [go], got: %s", lines[0])
	}
}

func TestCLI_FormatDisplayLabel(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%L: %p")
	if err != nil {
		t.Fatalf("pj --format display label failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if !strings.HasPrefix(lines[0], "Go: ") {
		t.Errorf("Output should start with 'Go: ', got: %s", lines[0])
	}
}

func TestCLI_FormatLabelAnsi(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--ansi", "--format", "%l %p")
	if err != nil {
		t.Fatalf("pj --format --ansi label failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Label should be wrapped in dim ANSI: \033[2m...\033[22m
	if !strings.Contains(lines[0], "\033[2m") {
		t.Errorf("Label should contain ANSI dim code, got: %q", lines[0])
	}
	if !strings.Contains(lines[0], "go\033[22m") {
		t.Errorf("Label should contain 'go' with dim reset, got: %q", lines[0])
	}
}

func TestCLI_FormatDisplayLabelAnsi(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--ansi", "--format", "%L %p")
	if err != nil {
		t.Fatalf("pj --format --ansi display label failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Display label should be wrapped in dim ANSI: \033[2m...\033[22m
	if !strings.Contains(lines[0], "\033[2mGo\033[22m") {
		t.Errorf("Display label should be dim-wrapped 'Go', got: %q", lines[0])
	}
}

func TestCLI_FormatColor(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%c %p")
	if err != nil {
		t.Fatalf("pj --format color failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if !strings.HasPrefix(lines[0], "cyan ") {
		t.Errorf("Output should start with 'cyan ', got: %s", lines[0])
	}
}

func TestCLI_FormatEscapePercent(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--format", "%%p: %p")
	if err != nil {
		t.Fatalf("pj --format escape failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	expected := "%p: " + filepath.Join(tmpDir, "project")
	if lines[0] != expected {
		t.Errorf("Output = %q, want %q", lines[0], expected)
	}
}

func TestCLI_FormatIgnoredWithJSON(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir, "project", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--json", "--format", "%n")
	if err != nil {
		t.Fatalf("pj --json --format failed: %v\nStderr: %s", err, stderr)
	}

	// Should still be valid JSON (--format ignored with --json)
	if !json.Valid([]byte(stdout)) {
		t.Errorf("Output should be valid JSON when --json is used, got: %s", stdout)
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

func TestSortProjects(t *testing.T) {
	mapper := icons.NewMapper(
		map[string]string{},
		map[string]string{},
		map[string]string{".git": "git", "go.mod": "go", "Cargo.toml": "rust"},
		map[string]string{},
	)

	projects := []discover.Project{
		{Path: "/z/project", Marker: "go.mod", Priority: 10},
		{Path: "/a/project", Marker: ".git", Priority: 1},
		{Path: "/m/project", Marker: "Cargo.toml", Priority: 10},
	}

	tests := []struct {
		name      string
		sortBy    string
		direction string
		expected  []string
	}{
		{
			name:      "alpha asc",
			sortBy:    "alpha",
			direction: "asc",
			expected:  []string{"/a/project", "/m/project", "/z/project"},
		},
		{
			name:      "alpha desc",
			sortBy:    "alpha",
			direction: "desc",
			expected:  []string{"/z/project", "/m/project", "/a/project"},
		},
		{
			name:      "priority asc",
			sortBy:    "priority",
			direction: "asc",
			expected:  []string{"/a/project", "/m/project", "/z/project"},
		},
		{
			name:      "priority desc",
			sortBy:    "priority",
			direction: "desc",
			expected:  []string{"/m/project", "/z/project", "/a/project"},
		},
		{
			name:      "label asc",
			sortBy:    "label",
			direction: "asc",
			expected:  []string{"/a/project", "/z/project", "/m/project"},
		},
		{
			name:      "label desc",
			sortBy:    "label",
			direction: "desc",
			expected:  []string{"/m/project", "/z/project", "/a/project"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := make([]discover.Project, len(projects))
			copy(p, projects)
			sortProjects(p, tt.sortBy, tt.direction, mapper)
			for i, proj := range p {
				if proj.Path != tt.expected[i] {
					t.Errorf("position %d: got %q, want %q", i, proj.Path, tt.expected[i])
				}
			}
		})
	}
}

func TestCLI_SortAlpha(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "z-project", "go.mod")
	createTestProject(t, tmpDir, "a-project", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--sort", "alpha", "--sort-direction", "asc")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "a-project") {
		t.Errorf("Alpha asc: first line should be a-project, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "z-project") {
		t.Errorf("Alpha asc: second line should be z-project, got: %s", lines[1])
	}
}

func TestCLI_SortPriorityAsc(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "go-project", "go.mod")
	createTestProject(t, tmpDir, "git-project", ".git/")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--sort", "priority", "--sort-direction", "asc")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(lines))
	}

	// Ascending: low priority (.git=1) first
	if !strings.Contains(lines[0], "git-project") {
		t.Errorf("Priority asc: first line should be git-project, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "go-project") {
		t.Errorf("Priority asc: second line should be go-project, got: %s", lines[1])
	}
}

func TestCLI_SortLabel(t *testing.T) {
	tmpDir := t.TempDir()

	// Labels: go.mod="go", .git="git", Cargo.toml="rust"
	createTestProject(t, tmpDir, "rust-project", "Cargo.toml")
	createTestProject(t, tmpDir, "git-project", ".git/")
	createTestProject(t, tmpDir, "go-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--sort", "label")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 projects, got %d", len(lines))
	}

	// Default direction is desc, so label desc: rust > go > git
	if !strings.Contains(lines[0], "rust-project") {
		t.Errorf("Label desc: first line should be rust-project, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "go-project") {
		t.Errorf("Label desc: second line should be go-project, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "git-project") {
		t.Errorf("Label desc: third line should be git-project, got: %s", lines[2])
	}
}

func TestCLI_SortAlphaDesc(t *testing.T) {
	tmpDir := t.TempDir()

	createTestProject(t, tmpDir, "a-project", ".git/")
	createTestProject(t, tmpDir, "z-project", "go.mod")

	stdout, stderr, err := runPJ(t, "-p", tmpDir, "--no-cache", "--sort", "alpha", "--sort-direction", "desc")
	if err != nil {
		t.Fatalf("pj failed: %v\nStderr: %s", err, stderr)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "z-project") {
		t.Errorf("Alpha desc: first line should be z-project, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "a-project") {
		t.Errorf("Alpha desc: second line should be a-project, got: %s", lines[1])
	}
}

func TestCLI_SortInvalidValue(t *testing.T) {
	_, _, err := runPJ(t, "--sort", "invalid")
	if err == nil {
		t.Error("Invalid sort value should produce an error")
	}
}

func TestCLI_SortDirectionInvalidValue(t *testing.T) {
	_, _, err := runPJ(t, "--sort-direction", "invalid")
	if err == nil {
		t.Error("Invalid sort-direction value should produce an error")
	}
}
