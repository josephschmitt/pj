package main

import (
	"fmt"
	"os"
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
	Icons      bool     `help:"Show marker-based icons"`
	Strip      bool     `help:"Strip icons from output"`
	IconMap    []string `help:"Override icon mapping (MARKER:ICON)"`
	NoCache    bool     `help:"Skip cache, force fresh search"`
	ClearCache bool     `help:"Clear cache and exit"`
	Verbose    bool     `short:"v" help:"Enable debug output"`
	Version    bool     `short:"V" help:"Show version"`
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

	// Load configuration
	cfg, err := config.Load(cli.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Merge CLI flags with config
	if err := cfg.MergeFlags(&cli); err != nil {
		fmt.Fprintf(os.Stderr, "Error merging config: %v\n", err)
		os.Exit(1)
	}

	if cli.Verbose {
		fmt.Fprintf(os.Stderr, "Config: %+v\n", cfg)
	}

	// Handle icon map overrides
	iconMapper := icons.NewMapper(cfg.Icons)
	if len(cli.IconMap) > 0 {
		for _, mapping := range cli.IconMap {
			parts := strings.SplitN(mapping, ":", 2)
			if len(parts) == 2 {
				iconMapper.Set(parts[0], parts[1])
			}
		}
	}

	// Initialize cache
	cacheManager := cache.New(cfg, cli.Verbose)

	// Handle cache clearing
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

	// Try cache first (unless --no-cache)
	if !cli.NoCache {
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

	// Discover projects if no cache hit
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

		// Cache results
		if err := cacheManager.Set(projects); err != nil && cli.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache results: %v\n", err)
		}
	}

	// Output projects
	for _, p := range projects {
		output := p.Path
		if cli.Icons && !cli.Strip {
			icon := iconMapper.Get(p.Marker)
			output = fmt.Sprintf("%s %s", icon, output)
		}
		fmt.Println(output)
	}

	ctx.Exit(0)
}
