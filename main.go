package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/josephschmitt/pj/internal/cache"
	"github.com/josephschmitt/pj/internal/config"
	"github.com/josephschmitt/pj/internal/discover"
	"github.com/josephschmitt/pj/internal/icons"
)

var version = "dev"

type CLI struct {
	Config     string   `short:"c" help:"Config file path" type:"path"`
	Path       []string `short:"p" help:"Add search path (repeatable)"`
	Marker     []string `short:"m" help:"Add project marker (repeatable)"`
	Exclude    []string `short:"e" help:"Exclude pattern (repeatable)"`
	MaxDepth   int      `short:"d" help:"Maximum search depth"`
	NoIgnore   bool     `help:"Don't respect .gitignore and .ignore files"`
	NoNested   bool     `help:"Don't search for projects inside other projects"`
	Icons      bool     `help:"Show marker-based icons"`
	Strip      bool     `help:"Strip icons from output"`
	IconMap    []string `help:"Override icon mapping (MARKER:ICON)"`
	Ansi       bool     `help:"Colorize icons with ANSI codes"`
	ColorMap   []string `help:"Override icon color (MARKER:COLOR)"`
	NoCache    bool     `help:"Skip cache, force fresh search"`
	ClearCache bool     `help:"Clear cache and exit"`
	JSON       bool     `short:"j" help:"Output results in JSON format"`
	Verbose    bool     `short:"v" help:"Enable debug output"`
	Version    bool     `short:"V" help:"Show version"`
}

func stdinIsPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func readPathsFromStdin(verbose bool) []string {
	var paths []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			continue
		}
		if strings.HasPrefix(path, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				path = filepath.Join(home, path[1:])
			}
		}
		if _, err := os.Stat(path); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "warning: skipping invalid path: %s\n", path)
			}
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("pj"),
		kong.Description("Fast project directory finder"),
		kong.UsageOnError(),
	)

	if cli.Version {
		fmt.Println("pj version", version)
		os.Exit(0)
	}

	cfg, err := config.LoadWithVerbose(cli.Config, cli.Verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.MergeFlags(&cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error merging config: %v\n", err)
		os.Exit(1)
	}

	stdinMode := false
	if stdinIsPiped() {
		stdinPaths := readPathsFromStdin(cli.Verbose)
		if len(stdinPaths) > 0 {
			cfg.SearchPaths = stdinPaths
			stdinMode = true
		}
	}

	if cli.Verbose {
		fmt.Fprintf(os.Stderr, "Config: %+v\n", cfg)
	}

	iconMapper := icons.NewMapper(cfg.GetIcons(), cfg.GetColors())
	if len(cli.IconMap) > 0 {
		for _, mapping := range cli.IconMap {
			parts := strings.SplitN(mapping, ":", 2)
			if len(parts) == 2 {
				iconMapper.Set(parts[0], parts[1])
			}
		}
	}
	if len(cli.ColorMap) > 0 {
		for _, mapping := range cli.ColorMap {
			parts := strings.SplitN(mapping, ":", 2)
			if len(parts) == 2 {
				iconMapper.SetColor(parts[0], parts[1])
			}
		}
	}

	cacheManager := cache.New(cfg, cli.Verbose)

	if cli.ClearCache {
		if err := cacheManager.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		if cli.Verbose {
			fmt.Fprintf(os.Stderr, "Cache cleared\n")
		}
		os.Exit(0)
	}

	var projects []discover.Project

	if !cli.NoCache && !stdinMode {
		cached, err := cacheManager.Get()
		if err == nil && cached != nil {
			if cli.Verbose {
				fmt.Fprintf(os.Stderr, "Using cached results (%d projects)\n", len(cached))
			}
			projects = cached
		} else if cli.Verbose && err != nil {
			fmt.Fprintf(os.Stderr, "Cache miss: %v\n", err)
		}
	}

	if projects == nil {
		discoverer := discover.New(cfg, cli.Verbose)
		projects, err = discoverer.Discover()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering projects: %v\n", err)
			os.Exit(1)
		}

		if cli.Verbose {
			fmt.Fprintf(os.Stderr, "Discovered %d projects\n", len(projects))
		}

		if !stdinMode {
			if err := cacheManager.Set(projects); err != nil && cli.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to cache results: %v\n", err)
			}
		}
	}

	if cli.JSON {
		type projectJSON struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Marker string `json:"marker"`
			Icon   string `json:"icon,omitempty"`
			Color  string `json:"color,omitempty"`
		}
		type outputJSON struct {
			Projects []projectJSON `json:"projects"`
		}

		jsonProjects := make([]projectJSON, len(projects))
		for i, p := range projects {
			icon := ""
			color := ""
			if cli.Icons {
				icon = iconMapper.Get(p.Marker)
				if cli.Ansi {
					color = iconMapper.GetColor(p.Marker)
				}
			}
			jsonProjects[i] = projectJSON{
				Path:   p.Path,
				Name:   filepath.Base(p.Path),
				Marker: p.Marker,
				Icon:   icon,
				Color:  color,
			}
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(outputJSON{Projects: jsonProjects}); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		for _, p := range projects {
			output := p.Path
			if cli.Icons && !cli.Strip {
				icon := iconMapper.Format(p.Marker, cli.Ansi)
				output = fmt.Sprintf("%s %s", icon, output)
			}
			fmt.Println(output)
		}
	}

	ctx.Exit(0)
}
