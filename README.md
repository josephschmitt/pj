<div align="center">
  <img src="docs/image.png" alt="pj logo" width="600">

  # pj - Project Finder CLI

  Fast project directory finder that searches your filesystem for git repositories and project directories. Built for speed and seamless integration with fuzzy finders like [fzf](https://github.com/junegunn/fzf) and [television](https://github.com/alexpasmantier/television).
</div>

<img width="1508" height="1126" alt="pj1" src="https://github.com/user-attachments/assets/c55db16e-68e0-4029-ae6f-f0ee014a7481" />
<img width="1508" height="1126" alt="pj2" src="https://github.com/user-attachments/assets/3e80cd9f-a03c-4260-be4a-52ef144a3d85" />

## Features

- **Fast Discovery**: Quickly finds all projects across multiple search paths
- **Smart Caching**: Caches results for instant subsequent searches (5-minute TTL)
- **Flexible Markers**: Detects projects by `.git`, `go.mod`, `package.json`, `Cargo.toml`, and more
- **Icon Support**: Display pretty icons for different project types (Nerd Fonts required)
- **ANSI Color Support**: Colorize icons with ANSI codes for terminal tools like `fzf` and `television`
- **Unix Pipeline Support**: Pipe paths in and results out - works seamlessly in command chains
- **Configurable**: YAML configuration file with sensible defaults
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Fuzzy Finder Integration**: Perfect for fuzzy finding and quick navigation with `fzf`, `television`, or your favorite fuzzy finder

## The PJ Cinematic Universe

The `pj` command was designed to be flexible and portable, meaning you can bring the power of `pj` to a number of projects and use cases:

- **[`pj.nvim`](https://github.com/josephschmitt/pj.nvim)**: Neovim integration with `pj` as a project picker. Supports all the major pickers, including Snacks, Telescope, mini.pick, fzf-lua, and Television.
- **[`pj-node`](https://github.com/josephschmitt/pj-node)**: NodeJS wrapper around `pj` binary, providing a node-based installation method as well as a TypeScript-native API.
- **[`pj-raycast`](https://github.com/josephschmitt/pj-raycast)**: Raycast extension for quickly navigating to your projects.

Have a project that integrates `pj`? Let me know and I'll add it to the list!

## Installation

### Homebrew (macOS/Linux)

```bash
brew install josephschmitt/tap/pj
```

### Nix (macOS/Linux)

```bash
# Try it out without installing
nix run github:josephschmitt/pj

# Install to profile
nix profile install github:josephschmitt/pj

# Add to your flake.nix
{
  inputs.pj.url = "github:josephschmitt/pj";
}
```

### Go Install (All Platforms)

```bash
go install github.com/josephschmitt/pj@latest
```

### gah (All Platforms)

```bash
# First, install gah: https://github.com/get-gah/gah
gah install josephschmitt/pj
```

### Scoop (Windows)

```powershell
scoop bucket add josephschmitt https://github.com/josephschmitt/scoop-bucket
scoop install pj
```

### Linux Packages

Download packages from the [releases page](https://github.com/josephschmitt/pj/releases). Available for both `amd64` and `arm64` architectures.

<!-- x-release-please-start-version -->

#### Debian/Ubuntu (.deb)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_amd64.deb
sudo dpkg -i pj_1.11.0_linux_amd64.deb

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_arm64.deb
sudo dpkg -i pj_1.11.0_linux_arm64.deb
```

#### Fedora/RHEL/CentOS (.rpm)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_amd64.rpm
sudo rpm -i pj_1.11.0_linux_amd64.rpm

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_arm64.rpm
sudo rpm -i pj_1.11.0_linux_arm64.rpm
```

#### Alpine (.apk)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_amd64.apk
apk add --allow-untrusted pj_1.11.0_linux_amd64.apk

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_arm64.apk
apk add --allow-untrusted pj_1.11.0_linux_arm64.apk
```

#### Arch Linux (.pkg.tar.zst)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_amd64.pkg.tar.zst
sudo pacman -U pj_1.11.0_linux_amd64.pkg.tar.zst

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_arm64.pkg.tar.zst
sudo pacman -U pj_1.11.0_linux_arm64.pkg.tar.zst
```

### Pre-built Binaries (All Platforms)

Download the latest binaries for your platform from the [releases page](https://github.com/josephschmitt/pj/releases).

Available platforms:
- **macOS**: `darwin_amd64` (Intel), `darwin_arm64` (Apple Silicon)
- **Linux**: `linux_amd64`, `linux_arm64`
- **Windows**: `windows_amd64`, `windows_arm64`

```bash
# Example: macOS Apple Silicon
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_darwin_arm64.tar.gz
tar -xzf pj_1.11.0_darwin_arm64.tar.gz
sudo mv pj /usr/local/bin/

# Example: Linux amd64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_amd64.tar.gz
tar -xzf pj_1.11.0_linux_amd64.tar.gz
sudo mv pj /usr/local/bin/

# Example: Linux arm64
wget https://github.com/josephschmitt/pj/releases/download/v1.11.0/pj_1.11.0_linux_arm64.tar.gz
tar -xzf pj_1.11.0_linux_arm64.tar.gz
sudo mv pj /usr/local/bin/
```

<!-- x-release-please-end -->

### From Source

```bash
git clone https://github.com/josephschmitt/pj.git
cd pj
make install
```

## Usage

### Basic Usage

```bash
# List all projects
pj

# Show projects with icons
pj --icons

# Show projects with colored icons
pj --icons --ansi

# Force fresh search (skip cache)
pj --no-cache

# Clear cache
pj --clear-cache

# Show version
pj --version
```

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config PATH` | `-c` | Config file path (default: `~/.config/pj/config.yaml`) |
| `--path PATH` | `-p` | Add search path (repeatable) |
| `--marker MARKER` | `-m` | Add project marker (repeatable) |
| `--exclude PATTERN` | `-e` | Exclude pattern (repeatable) |
| `--max-depth N` | `-d` | Maximum search depth |
| `--icons` | | Show marker-based icons |
| `--ansi` | | Colorize icons with ANSI codes |
| `--strip` | | Strip icons from output |
| `--icon-map MARKER:ICON` | | Override icon mapping |
| `--color-map MARKER:COLOR` | | Override icon color |
| `--no-cache` | | Skip cache, force fresh search |
| `--clear-cache` | | Clear cache and exit |
| `--verbose` | `-v` | Enable debug output |
| `--version` | `-V` | Show version |

### Examples

```bash
# Search additional paths
pj -p ~/personal -p ~/work

# Add custom project marker
pj -m requirements.txt

# Exclude directories
pj -e tmp -e cache

# Custom icon mapping
pj --icons --icon-map "go.mod:🐹"

# Colored icons with per-marker override
pj --icons --ansi --color-map "go.mod:cyan"

# Verbose output for debugging
pj -v
```

### Unix Pipeline Support

`pj` follows the Unix philosophy and can be used as both a filter and a data source in pipelines. When paths are piped into `pj` via stdin, it automatically detects this and searches only those paths (bypassing cache for dynamic results).

#### Piping Into pj

Filter a list of directories to find projects within them:

```bash
# Find projects in specific directories
ls -d ~/work/*/ | pj --icons

# Use with find to search specific locations
find ~/code -maxdepth 1 -type d | pj

# Combine multiple directory sources
echo -e "~/personal\n~/work\n~/experiments" | pj --icons

# Filter projects from a specific subdirectory pattern
ls -d ~/repos/*/src | pj
```

#### Piping Out of pj

Use `pj` output as input for other Unix tools:

```bash
# Count total projects in a directory tree
ls -d ~/development/*/ | pj | wc -l

# Search project paths by name
pj | grep -i "react"

# Filter projects by path location
pj | grep "/work/"

# Create a backup script for all projects
pj | xargs -I {} tar -czf {}.tar.gz {}

# Check git status for all projects
pj | xargs -I {} sh -c 'echo "=== {} ===" && git -C {} status -s'

# Find projects modified in the last 7 days
pj | xargs -I {} find {} -maxdepth 1 -mtime -7 -type f

# Open a random project (requires function wrapper, not direct cd)
cd "$(pj | shuf -n 1)"
```

#### Combining Both

Chain `pj` with other tools for powerful workflows:

```bash
# Count projects in specific directories
ls -d ~/code/*/ | pj | wc -l

# Get total size of all your projects
pj | xargs du -sh

# Find all projects that have uncommitted changes
pj | xargs -I {} sh -c 'git -C {} diff --quiet || echo {}'

# List projects sorted by last modification time
pj | xargs ls -dt
```

**Note:** When using stdin, `pj` automatically skips the cache since piped input is dynamic. Invalid paths are silently ignored (use `-v` to see warnings).

## Configuration

`pj` reads configuration from `~/.config/pj/config.yaml` (or `$XDG_CONFIG_HOME/pj/config.yaml`).

### Example Config

```yaml
# Paths to search for projects
search_paths:
  - ~/projects
  - ~/code
  - ~/development
  - ~/work

# Files/directories that mark a project (with optional icons, colors, and priority)
# Each marker can be a simple string or an object with marker, icon, color, and priority fields
# Priority determines which marker is used when multiple exist in the same directory
# Higher priority wins. Default priorities: language=10, infrastructure=7, IDE=5, generic=1
# Glob patterns are supported: *.csproj, *.sln, test_*.yml, etc.
# Color is used with --ansi flag. Available colors: black, red, green, yellow, blue,
# magenta, cyan, white, and bright- variants. Default: blue
markers:
  - marker: .git
    icon: ""      # \ue65d - Git icon
  - marker: go.mod
    icon: "󰟓"      # \U000f07d3 - Go icon
  - marker: package.json
    icon: "󰎙"      # \U000f0399 - npm icon
  - marker: Cargo.toml
    icon: ""      # \ue68b - Rust icon
  - marker: pyproject.toml
    icon: ""      # \ue606 - Python icon
  - marker: Makefile
    icon: ""      # \ue673 - Makefile icon
  - marker: flake.nix
    icon: ""      # \ue843 - Nix icon
  - marker: .vscode
    icon: "󰨞"      # \U000f0a1e - VS Code icon
  - marker: .idea
    icon: ""      # \ue7b5 - IntelliJ icon
  - marker: build.gradle
    # icon, color, and priority are optional - omit for defaults
  - marker: "*.csproj"       # Glob patterns for variable file names
    icon: "󰪮"
  - marker: "*.sln"
    icon: "󰪮"

# Maximum directory depth to search
max_depth: 3

# Patterns to exclude from search
excludes:
  - node_modules
  - .terraform
  - vendor
  - .git
  - target
  - dist
  - build
  - .next
  - .nuxt

# Cache TTL in seconds (default: 300 = 5 minutes)
cache_ttl: 300
```

#### Legacy Format (Deprecated)

The old format with separate `markers` and `icons` fields is still supported for backward compatibility:

```yaml
# Old format (deprecated)
markers:
  - .git
  - go.mod
  - package.json

icons:
  .git: ""
  go.mod: "󰟓"
  package.json: "󰎙"
```

If both formats are used, the new format takes precedence. Run with `-v` to see deprecation warnings.

#### Marker Priority

When a directory contains multiple markers (e.g., both `.git` and `go.mod`), the marker with the highest priority is used. This affects which icon is displayed and how projects are sorted.

Default priority tiers:
- **10** - Language-specific: `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `flake.nix`
- **7** - Infrastructure: `Dockerfile`
- **5** - IDE markers: `.vscode`, `.idea`, `.fleet`, `.zed`, `.project`
- **1** - Generic: `.git`, `Makefile`

You can customize priority for any marker:

```yaml
markers:
  - marker: .git
    priority: 100  # Make .git highest priority
  - marker: my-custom-marker
    priority: 15   # Custom marker with custom priority
```

#### Glob Pattern Markers

Markers support glob patterns (`*`, `?`, `[]`) for detecting projects with variable file names. This is useful for ecosystems like .NET where project files have names like `MyApp.csproj` or `Solution.sln`.

```yaml
markers:
  - marker: "*.csproj"    # Matches any .csproj file
    icon: "󰪮"
    priority: 10
  - marker: "*.sln"       # Matches any .sln file
    icon: "󰪮"
    priority: 12
  - marker: "test_*.yml"  # Matches test_config.yml, test_data.yml, etc.
  - marker: "config[0-9].json"  # Matches config1.json, config2.json, etc.
```

When a glob pattern matches, the actual matched filename is used as the marker (e.g., `MyApp.csproj` instead of `*.csproj`). If multiple files match the same pattern, the first alphabetically is used.

Exact markers (like `.git`, `go.mod`) are checked first using fast `os.Stat` calls. Pattern markers are checked by reading directory contents, so they have slightly more overhead but are still efficient.

### Config Priority

CLI flags override config file settings, which override defaults.

```bash
# Config says max_depth: 3, this overrides to 5
pj -d 5
```

## Integrations

The real power of `pj` comes from integrating it with other tools for quick project navigation.

### Neovim

[pj.nvim](https://github.com/josephschmitt/pj.nvim) provides native Neovim integration with support for multiple pickers including Snacks, Telescope, fzf-lua, Television, and mini.pick. Features include automatic binary installation, session manager integration, and consistent keybindings across all picker implementations.

### Shell (Bash/Zsh)

Add to your `~/.bashrc` or `~/.zshrc`:

#### Using fzf

```bash
# Quick project navigation with fzf
pjf() {
  local project
  project=$(pj --icons --ansi | fzf --ansi --preview 'ls -la {2}' --preview-window right:60%) &&
  cd "$(echo "$project" | awk '{print $2}')"
}
```

Or for a simpler version without icons:

```bash
pjf() {
  local project
  project=$(pj | fzf --preview 'ls -la {}' --preview-window right:60%) &&
  cd "$project"
}
```

#### Using television

```bash
# Quick project navigation with television
pjt() {
  local project
  project=$(pj --icons --ansi | tv) &&
  cd "$(echo "$project" | awk '{print $2}')"
}
```

Or without icons:

```bash
pjt() {
  local project
  project=$(pj | tv) &&
  cd "$project"
}
```

### Fish Shell

Add to your `~/.config/fish/config.fish`:

#### Using fzf

```fish
function pjf
    set -l project (pj --icons --ansi | fzf --ansi --preview 'ls -la (echo {} | awk \'{print $2}\')' --preview-window right:60%)
    and cd (echo $project | awk '{print $2}')
end
```

#### Using television

```fish
function pjt
    set -l project (pj --icons --ansi | tv)
    and cd (echo $project | awk '{print $2}')
end
```

### Tmux Integration

Create a new tmux session in a project:

```bash
pjt() {
  local project
  project=$(pj | fzf --preview 'ls -la {}' --preview-window right:60%)
  [ -n "$project" ] && tmux new-session -A -s "$(basename "$project")" -c "$project"
}
```


## How It Works

1. **First Run**: `pj` searches configured paths for project markers, caches results
2. **Subsequent Runs**: Returns cached results instantly (5-minute TTL by default)
3. **Cache Invalidation**: Cache expires after TTL or can be cleared with `--clear-cache`
4. **Smart Exclusion**: Automatically skips `node_modules`, `vendor`, etc. to speed up search

## Performance

- Initial scan (no cache): ~100-500ms for typical setups
- Cached results: <10ms
- Handles thousands of projects efficiently

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build instructions, testing guidelines, and our commit message conventions.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Joseph Schmitt ([@josephschmitt](https://github.com/josephschmitt))

## Related Projects

- [gah](https://github.com/get-gah/gah) - GitHub Asset Helper for installing binaries
- [z](https://github.com/rupa/z) - Jump around directories
- [autojump](https://github.com/wting/autojump) - Fast way to navigate filesystem

---

**Note**: Icon display requires a [Nerd Font](https://www.nerdfonts.com/) to be installed and used in your terminal.
