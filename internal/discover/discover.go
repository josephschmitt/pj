package discover

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/josephschmitt/pj/internal/config"
)

// Project represents a discovered project directory
type Project struct {
	Path           string `json:"path"`
	Marker         string `json:"marker"`
	Priority       int    `json:"priority"`
	IsWorktree     bool   `json:"isWorktree,omitempty"`
	WorktreeParent string `json:"worktreeParent,omitempty"`
}

// Discoverer handles project discovery
type Discoverer struct {
	config  *config.Config
	verbose bool
}

// New creates a new Discoverer
func New(cfg *config.Config, verbose bool) *Discoverer {
	cfg.EnsureMarkerCategories()
	return &Discoverer{
		config:  cfg,
		verbose: verbose,
	}
}

// Marker specificity rankings (higher = more specific)
var markerSpecificity = map[string]int{
	".git":           1,
	"Makefile":       1,
	"package.json":   7,
	"Cargo.toml":     10,
	"go.mod":         10,
	"pyproject.toml": 10,
	"flake.nix":      3,
	"pom.xml":        10,
	"build.gradle":   10,
	"CMakeLists.txt": 10,
	".vscode":        5,
	".idea":          5,
	".fleet":         5,
	".project":       5,
	".zed":           5,
	"Dockerfile":     7,
}

// Discover finds all project directories
func (d *Discoverer) Discover() ([]Project, error) {
	var wg sync.WaitGroup
	results := make(chan Project, 100)

	// Fan-out: one goroutine per search path
	for _, root := range d.config.SearchPaths {
		if len(root) > 0 && root[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			root = filepath.Join(home, root[1:])
		}

		if _, err := os.Stat(root); os.IsNotExist(err) {
			if d.verbose {
				fmt.Fprintf(os.Stderr, "Skipping non-existent path: %s\n", root)
			}
			continue
		}

		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			if d.verbose {
				fmt.Fprintf(os.Stderr, "Searching %s...\n", root)
			}
			d.walkPath(root, results)
		}(root)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	seen := make(map[string]bool)
	var projects []Project
	for p := range results {
		if !seen[p.Path] {
			seen[p.Path] = true
			projects = append(projects, p)
		}
	}

	// Sort by path for deterministic output; presentation sorting is handled by the caller
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Path < projects[j].Path
	})

	return projects, nil
}

// walkPath walks a single search path
func (d *Discoverer) walkPath(root string, results chan<- Project) {
	baseDepth := strings.Count(root, string(os.PathSeparator))

	ignoreFileNames := []string{".gitignore", ".ignore"}
	ignoreStack := NewIgnoreStack(!d.config.NoIgnore, ignoreFileNames)

	previousDepth := -1

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip paths we can't access
		}

		if !entry.IsDir() {
			return nil
		}

		currentDepth := strings.Count(path, string(os.PathSeparator)) - baseDepth

		if previousDepth >= 0 && currentDepth <= previousDepth {
			ignoreStack.Leave(currentDepth)
		}
		previousDepth = currentDepth

		if ignoreStack.ShouldIgnore(path, true) {
			return fs.SkipDir
		}

		if err := ignoreStack.Enter(path, currentDepth); err != nil {
			if d.verbose {
				fmt.Fprintf(os.Stderr, "Error loading ignore files in %s: %v\n", path, err)
			}
		}

		if currentDepth > d.config.MaxDepth {
			return fs.SkipDir
		}

		dirName := filepath.Base(path)
		for _, exclude := range d.config.Excludes {
			if matchPattern(dirName, exclude) {
				return fs.SkipDir
			}
		}

		// Check for project markers - find the highest priority marker
		bestMarker, bestPriority := d.findBestMarker(path)

		// If we found any marker, emit the project with the best one
		if bestMarker != "" {
			project := Project{
				Path:     path,
				Marker:   bestMarker,
				Priority: bestPriority,
			}

			// Path A: detect if this is a worktree (.git is a file, not a directory)
			gitPath := filepath.Join(path, ".git")
			if info, err := os.Lstat(gitPath); err == nil && !info.IsDir() {
				parent := parseWorktreeGitFile(gitPath)
				if parent != "" {
					if d.config.NoWorktrees {
						return fs.SkipDir
					}
					project.IsWorktree = true
					project.WorktreeParent = parent
				}
			}

			results <- project

			// Path B: discover linked worktrees from parent repos
			if d.config.Worktrees && !project.IsWorktree {
				d.discoverWorktrees(path, results)
			}

			// Skip subdirectories unless nested discovery is enabled
			if !d.config.Nested {
				return fs.SkipDir
			}
		}

		return nil
	})

	if err != nil && d.verbose {
		fmt.Fprintf(os.Stderr, "Error walking %s: %v\n", root, err)
	}
}

// getMarkerPriority returns the priority for a marker, checking config first, then defaults
func (d *Discoverer) getMarkerPriority(marker string) int {
	priority := d.config.Priorities[marker]
	if priority == 0 {
		priority = markerSpecificity[marker]
	}
	if priority == 0 {
		priority = 1 // Default priority for unknown markers
	}
	return priority
}

// checkPatternMarkers checks pattern-based markers by reading directory contents once
func (d *Discoverer) checkPatternMarkers(dir string) (string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0
	}

	var bestMatch string
	var bestPriority int

	for _, pattern := range d.config.PatternMarkers {
		for _, entry := range entries {
			if entry.IsDir() {
				continue // Skip directories for file patterns
			}
			matched, _ := filepath.Match(pattern, entry.Name())
			if matched {
				priority := d.getMarkerPriority(pattern)
				if priority > bestPriority {
					bestMatch = entry.Name()
					bestPriority = priority
				}
				break // First match per pattern wins
			}
		}
	}
	return bestMatch, bestPriority
}

// parseWorktreeGitFile reads a .git file (not directory) and resolves the parent repo path.
// Worktree .git files contain "gitdir: <path>" pointing to the parent's .git/worktrees/<name>/.
func parseWorktreeGitFile(gitFilePath string) string {
	f, err := os.Open(gitFilePath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "gitdir: ") {
		return ""
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")

	// Resolve relative paths against the worktree directory
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(gitFilePath), gitdir)
	}
	gitdir = filepath.Clean(gitdir)

	// gitdir points to e.g. /parent/.git/worktrees/<name>
	// Walk up to find the parent repo: strip /.git/worktrees/<name> to get /parent
	parts := strings.Split(gitdir, string(os.PathSeparator))
	for i := len(parts) - 1; i >= 1; i-- {
		if parts[i] == "worktrees" && parts[i-1] == ".git" {
			parentPath := strings.Join(parts[:i-1], string(os.PathSeparator))
			if parentPath == "" {
				parentPath = "/"
			}
			return parentPath
		}
	}
	return ""
}

// discoverWorktrees finds git worktrees linked from a parent repo's .git/worktrees/ directory.
func (d *Discoverer) discoverWorktrees(repoPath string, results chan<- Project) {
	worktreesDir := filepath.Join(repoPath, ".git", "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return // No worktrees directory
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gitdirPath := filepath.Join(worktreesDir, entry.Name(), "gitdir")
		data, err := os.ReadFile(gitdirPath)
		if err != nil {
			if d.verbose {
				fmt.Fprintf(os.Stderr, "Warning: couldn't read gitdir for worktree %s: %v\n", entry.Name(), err)
			}
			continue
		}

		wtGitFile := strings.TrimSpace(string(data))

		// Resolve relative paths
		if !filepath.IsAbs(wtGitFile) {
			wtGitFile = filepath.Join(worktreesDir, entry.Name(), wtGitFile)
		}
		wtGitFile = filepath.Clean(wtGitFile)

		// The gitdir file contains the path to the worktree's .git file
		// The worktree root is its parent directory
		wtPath := wtGitFile
		if strings.HasSuffix(wtPath, string(os.PathSeparator)+".git") {
			wtPath = filepath.Dir(wtPath)
		}

		if _, err := os.Stat(wtPath); err != nil {
			if d.verbose {
				fmt.Fprintf(os.Stderr, "Warning: worktree path doesn't exist: %s\n", wtPath)
			}
			continue
		}

		// Check excludes
		wtName := filepath.Base(wtPath)
		excluded := false
		for _, exclude := range d.config.Excludes {
			if matchPattern(wtName, exclude) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Find the best marker in the worktree directory
		bestMarker, bestPriority := d.findBestMarker(wtPath)
		if bestMarker == "" {
			bestMarker = ".git"
			bestPriority = d.getMarkerPriority(".git")
		}

		results <- Project{
			Path:           wtPath,
			Marker:         bestMarker,
			Priority:       bestPriority,
			IsWorktree:     true,
			WorktreeParent: repoPath,
		}

		if d.verbose {
			fmt.Fprintf(os.Stderr, "Found worktree: %s (parent: %s)\n", wtPath, repoPath)
		}
	}
}

// findBestMarker checks a directory for configured markers and returns the best one.
func (d *Discoverer) findBestMarker(dir string) (string, int) {
	var bestMarker string
	var bestPriority int

	for _, marker := range d.config.ExactMarkers {
		markerPath := filepath.Join(dir, marker)
		if _, err := os.Stat(markerPath); err == nil {
			priority := d.getMarkerPriority(marker)
			if priority > bestPriority {
				bestMarker = marker
				bestPriority = priority
			}
		}
	}

	if len(d.config.PatternMarkers) > 0 {
		patternMarker, patternPriority := d.checkPatternMarkers(dir)
		if patternPriority > bestPriority {
			bestMarker = patternMarker
			bestPriority = patternPriority
		}
	}

	return bestMarker, bestPriority
}

// matchPattern checks if a name matches a pattern (simple glob support)
func matchPattern(name, pattern string) bool {
	if name == pattern {
		return true
	}

	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(name, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}

	matched, err := filepath.Match(pattern, name)
	if err == nil && matched {
		return true
	}

	return false
}
