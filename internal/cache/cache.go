package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/josephschmitt/pj/internal/config"
	"github.com/josephschmitt/pj/internal/discover"
)

// Manager handles caching of discovery results
type Manager struct {
	config  *config.Config
	verbose bool
	cacheDir string
}

// New creates a new cache manager
func New(cfg *config.Config, verbose bool) *Manager {
	cacheDir := getCacheDir()
	return &Manager{
		config:   cfg,
		verbose:  verbose,
		cacheDir: cacheDir,
	}
}

// Get retrieves cached projects if valid
func (m *Manager) Get() ([]discover.Project, error) {
	cachePath := m.getCachePath()

	info, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("cache not found")
	}
	if err != nil {
		return nil, err
	}

	age := time.Since(info.ModTime()).Seconds()
	if age > float64(m.config.CacheTTL) {
		if m.verbose {
			fmt.Fprintf(os.Stderr, "Cache expired (age: %.0fs, ttl: %ds)\n", age, m.config.CacheTTL)
		}
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var projects []discover.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// Set caches discovery results
func (m *Manager) Set(projects []discover.Project) error {
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return err
	}

	cachePath := m.getCachePath()

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// Clear removes all cache files
func (m *Manager) Clear() error {
	return os.RemoveAll(m.cacheDir)
}

// getCachePath returns the cache file path based on config hash
func (m *Manager) getCachePath() string {
	hash := m.computeConfigHash()
	return filepath.Join(m.cacheDir, fmt.Sprintf("cache-%s.json", hash))
}

// computeConfigHash creates a hash of the configuration
func (m *Manager) computeConfigHash() string {
	h := sha256.New()

	paths := make([]string, len(m.config.SearchPaths))
	copy(paths, m.config.SearchPaths)
	sort.Strings(paths)
	h.Write([]byte(strings.Join(paths, "|")))

	markers := make([]string, len(m.config.Markers))
	copy(markers, m.config.Markers)
	sort.Strings(markers)
	h.Write([]byte(strings.Join(markers, "|")))

	excludes := make([]string, len(m.config.Excludes))
	copy(excludes, m.config.Excludes)
	sort.Strings(excludes)
	h.Write([]byte(strings.Join(excludes, "|")))

	h.Write([]byte(fmt.Sprintf("%d", m.config.MaxDepth)))

	h.Write([]byte(fmt.Sprintf("%t", m.config.NoIgnore)))

	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// getCacheDir returns the cache directory using XDG_CACHE_HOME
func getCacheDir() string {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		cacheHome = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheHome, "pj")
}
