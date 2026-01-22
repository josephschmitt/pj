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
	return &Discoverer{
		config:  cfg,
		verbose: verbose,
	}
}

// Marker specificity rankings (higher = more specific)
var markerSpecificity = map[string]int{
	".git":           1,
	"Makefile":       1,
	"package.json":   10,
	"Cargo.toml":     10,
	"go.mod":         10,
	"pyproject.toml": 10,
	"flake.nix":      10,
	"pom.xml":        10,
	"build.gradle":   10,
	"CMakeLists.txt": 10,
}

// Discover finds all project directories
func (d *Discoverer) Discover() ([]Project, error) {
	var wg sync.WaitGroup
	results := make(chan Project, 100)

	// Fan-out: one goroutine per search path
	for _, root := range d.config.SearchPaths {
		// Expand home directory
		if len(root) > 0 && root[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			root = filepath.Join(home, root[1:])
		}

		// Check if path exists
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

	// Close channel when all walkers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and deduplicate
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

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip paths we can't access
		}

		if !entry.IsDir() {
			return nil
		}

		// Check max depth
		currentDepth := strings.Count(path, string(os.PathSeparator)) - baseDepth
		if currentDepth > d.config.MaxDepth {
			return fs.SkipDir
		}

		// Check excludes
		dirName := filepath.Base(path)
		for _, exclude := range d.config.Excludes {
			if matchPattern(dirName, exclude) {
				return fs.SkipDir
			}
		}

		// Check for project markers
		for _, marker := range d.config.Markers {
			markerPath := filepath.Join(path, marker)
			if _, err := os.Stat(markerPath); err == nil {
				priority := markerSpecificity[marker]
				if priority == 0 {
					priority = 1 // Default priority for unmarked markers
				}

				results <- Project{
					Path:     path,
					Marker:   marker,
					Priority: priority,
				}

				// Found a project, skip its subdirectories
				return fs.SkipDir
			}
		}

		return nil
	})

	if err != nil && d.verbose {
		fmt.Fprintf(os.Stderr, "Error walking %s: %v\n", root, err)
	}
}

// matchPattern checks if a name matches a pattern (simple glob support)
func matchPattern(name, pattern string) bool {
	// Simple exact match for now
	if name == pattern {
		return true
	}

	// Simple prefix/suffix matching
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(name, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}

	// Try filepath.Match for glob patterns
	matched, err := filepath.Match(pattern, name)
	if err == nil && matched {
		return true
	}

	return false
}
