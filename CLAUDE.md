# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`pj` is a fast project directory finder CLI written in Go that searches filesystems for git repositories and project directories. It's designed for speed and seamless integration with fuzzy finders like `fzf`.

## Build Commands

```bash
# Build for current platform
make build

# Install to $GOPATH/bin
make install

# Run the binary
make run

# Run tests with coverage
make test

# Run tests with race detector and detailed coverage
make test-coverage

# Run linter
make lint

# Cross-compile for all platforms (outputs to dist/)
make build-all

# Validate GoReleaser config
make release-check

# Test GoReleaser locally without publishing
make release-local

# Clean build artifacts
make clean
```

## Architecture

### Core Data Flow

1. **main.go** - CLI entry point using `kong` for argument parsing
2. **config package** - Loads YAML config from `~/.config/pj/config.yaml` or XDG_CONFIG_HOME, merges with CLI flags
3. **cache package** - Manages JSON cache files in `~/.cache/pj/` or XDG_CACHE_HOME with TTL-based invalidation
4. **discover package** - Walks filesystem concurrently to find project directories
5. **icons package** - Maps project markers (`.git`, `go.mod`, etc.) to Nerd Font icons

### Key Design Patterns

**Concurrent Discovery**: The `discover` package uses a fan-out goroutine pattern - one goroutine per search path, all feeding results into a shared channel. This provides significant speedup when searching multiple root directories.

**Config-Based Cache Keys**: Cache files are named using a SHA256 hash of the configuration (search paths, markers, excludes, max depth). This means different configurations automatically get separate cache files, preventing stale results when settings change.

**Priority-Based Sorting**: Projects are sorted by marker specificity (e.g., `package.json` has priority 10, `.git` has priority 1), then alphabetically. This ensures language-specific projects appear before generic git repos.

**Early Termination**: When a project marker is found, that directory subtree is skipped (returns `fs.SkipDir`). This prevents redundant scanning and respects that a parent project marker takes precedence over child markers.

## Configuration System

The config loading order is:
1. Defaults (defined in `config.defaults()`)
2. YAML file (`~/.config/pj/config.yaml`)
3. CLI flags (highest priority)

CLI flags use reflection to merge into config struct, avoiding tight coupling between CLI and config packages.

## Release Process

This project uses **Conventional Commits** for automated releases:

```bash
# Commit format
feat: new feature      # Minor version bump (0.1.0 → 0.2.0)
fix: bug fix          # Patch version bump (0.1.0 → 0.1.1)
feat!: breaking       # Major version bump (0.1.0 → 1.0.0)
docs: documentation   # No version bump
```

**Automated release workflow:**
1. Push commits with conventional format to `main`
2. `release-please` analyzes commits and creates/updates a Release PR
3. Merge the Release PR to trigger tag creation
4. Tag triggers GoReleaser to build and publish:
   - GitHub release with binaries for 6 platforms (darwin/linux/windows × amd64/arm64)
   - Homebrew formula to `josephschmitt/tap`
   - Scoop manifest to `josephschmitt/scoop-bucket`
   - Linux packages (.deb, .rpm, .apk, .pkg.tar.zst)

**Do not manually create tags.** The release-please workflow handles version management.

## Distribution

GoReleaser configuration (`.goreleaser.yaml`) handles multi-platform builds with:
- CGO disabled for static binaries
- Version injected via ldflags: `-X main.version={{.Version}}`
- Archives include README and LICENSE
- Homebrew formula auto-updates via GitHub token
- Linux packages (deb/rpm/apk/Arch) generated for both amd64 and arm64

The project also includes a Nix flake (`flake.nix`) for declarative distribution.

## Code Organization

```
pj/
├── main.go                    # CLI entry, orchestrates config/cache/discover/icons
├── internal/
│   ├── cache/                 # TTL-based JSON caching with config-hash keys
│   ├── config/                # YAML config loading + CLI flag merging
│   ├── discover/              # Concurrent filesystem walking with fs.WalkDir
│   └── icons/                 # Marker → Nerd Font icon mapping
├── .goreleaser.yaml           # Multi-platform release automation
└── .github/workflows/
    ├── ci.yml                 # Test/lint on push
    ├── release.yml            # GoReleaser on tag push
    └── release-please.yml     # Auto-generate release PRs from commits
```

## Important Implementation Details

**Version Management**: The `version` variable in `main.go` is set via ldflags at build time. Default is `"dev"` for local builds.

**Cache Invalidation**: Cache files are stored as `cache-<hash>.json` where hash represents the config. Changing any config parameter (paths, markers, excludes, depth) results in a different hash and thus a fresh search.

**XDG Compliance**: Config uses `XDG_CONFIG_HOME` (defaults to `~/.config`), cache uses `XDG_CACHE_HOME` (defaults to `~/.cache`).

**Exclusion Patterns**: The `matchPattern` function in `discover` package supports:
- Exact match: `node_modules`
- Prefix: `*_cache`
- Suffix: `.tmp*`
- Glob: `test_*_tmp`

**Marker Specificity Map**: When multiple markers exist in the same directory, the one with highest specificity is used. Language-specific markers (10) rank higher than generic markers (1).

## Dependencies

- `github.com/alecthomas/kong` - CLI argument parsing with struct tags
- `gopkg.in/yaml.v3` - YAML configuration file parsing

No external dependencies for core functionality (filesystem walking, caching, discovery).
