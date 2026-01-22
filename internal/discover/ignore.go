package discover

import (
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

// IgnoreStack manages hierarchical ignore patterns from .gitignore and .ignore files.
// It maintains a stack of ignore matchers as we traverse the directory tree.
type IgnoreStack struct {
	stack     []*ignoreEntry
	enabled   bool
	fileNames []string // e.g., [".gitignore", ".ignore"]
}

// ignoreEntry represents an ignore matcher at a specific directory depth.
type ignoreEntry struct {
	depth   int
	matcher *ignore.GitIgnore
	dir     string
}

// NewIgnoreStack creates a new ignore stack.
// If enabled is false, no ignore files will be respected.
// fileNames is the list of ignore file names to look for (e.g., ".gitignore", ".ignore").
func NewIgnoreStack(enabled bool, fileNames []string) *IgnoreStack {
	return &IgnoreStack{
		stack:     make([]*ignoreEntry, 0),
		enabled:   enabled,
		fileNames: fileNames,
	}
}

// Enter should be called when entering a directory during traversal.
// It checks for ignore files in the directory and adds them to the stack.
func (is *IgnoreStack) Enter(dir string, depth int) error {
	if !is.enabled {
		return nil
	}

	for _, fileName := range is.fileNames {
		ignoreFilePath := filepath.Join(dir, fileName)
		if _, err := os.Stat(ignoreFilePath); err == nil {
			matcher, err := ignore.CompileIgnoreFile(ignoreFilePath)
			if err != nil {
				// Silently skip malformed ignore files
				continue
			}

			is.stack = append(is.stack, &ignoreEntry{
				depth:   depth,
				matcher: matcher,
				dir:     dir,
			})
		}
	}

	return nil
}

// Leave should be called when ascending above a directory depth.
// It removes ignore entries that are deeper than the current depth.
func (is *IgnoreStack) Leave(depth int) {
	if !is.enabled {
		return
	}

	for len(is.stack) > 0 && is.stack[len(is.stack)-1].depth >= depth {
		is.stack = is.stack[:len(is.stack)-1]
	}
}

// ShouldIgnore checks if a path should be ignored based on active ignore rules.
// path should be an absolute path, and isDir indicates if it's a directory.
func (is *IgnoreStack) ShouldIgnore(path string, isDir bool) bool {
	if !is.enabled || len(is.stack) == 0 {
		return false
	}

	// Check against all active matchers (from innermost to outermost)
	// Process in reverse order so that more specific (deeper) rules take precedence
	for i := len(is.stack) - 1; i >= 0; i-- {
		entry := is.stack[i]

		relPath, err := filepath.Rel(entry.dir, path)
		if err != nil {
			// If we can't make it relative, skip this matcher
			continue
		}

		relPath = filepath.ToSlash(relPath)

		// For directories, append trailing slash for proper matching
		// This ensures patterns like "dir/" match correctly
		if isDir && relPath != "." {
			relPath = relPath + "/"
		}

		if entry.matcher.MatchesPath(relPath) {
			return true
		}
	}

	return false
}
