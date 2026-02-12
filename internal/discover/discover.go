package discover

import (
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
	Path     string
	Marker   string
	Priority int
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

	// Sort by priority (higher first), then by path
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Priority != projects[j].Priority {
			return projects[i].Priority > projects[j].Priority
		}
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
		// Phase 1: Check exact markers using os.Stat (fast path)
		var bestMarker string
		var bestPriority int
		for _, marker := range d.config.ExactMarkers {
			markerPath := filepath.Join(path, marker)
			if _, err := os.Stat(markerPath); err == nil {
				priority := d.getMarkerPriority(marker)
				if priority > bestPriority {
					bestMarker = marker
					bestPriority = priority
				}
			}
		}

		// Phase 2: Check pattern markers using directory listing (only if configured)
		if len(d.config.PatternMarkers) > 0 {
			patternMarker, patternPriority := d.checkPatternMarkers(path)
			if patternPriority > bestPriority {
				bestMarker = patternMarker
				bestPriority = patternPriority
			}
		}

		// If we found any marker, emit the project with the best one
		if bestMarker != "" {
			results <- Project{
				Path:     path,
				Marker:   bestMarker,
				Priority: bestPriority,
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
