<div align="center">
  <img src="docs/image.png" alt="pj logo" width="600">

  # pj - Project Finder CLI

  Fast project directory finder that searches your filesystem for git repositories and project directories. Built for speed and seamless integration with fuzzy finders like [fzf](https://github.com/junegunn/fzf) and [television](https://github.com/alexpasmantier/television).
</div>
￼
<img width="1187" height="842" alt="image" src="https://github.com/user-attachments/assets/009f465a-7a5a-4cea-9bb0-0f723591ea18" />
<img width="1768" height="1167" alt="image" src="https://github.com/user-attachments/assets/8c8c4e94-0752-4cd4-8684-c2207830a873" />

## Features

- **Fast Discovery**: Quickly finds all projects across multiple search paths
- **Smart Caching**: Caches results for instant subsequent searches (5-minute TTL)
- **Flexible Markers**: Detects projects by `.git`, `go.mod`, `package.json`, `Cargo.toml`, and more
- **Icon Support**: Display pretty icons for different project types (Nerd Fonts required)
- **Unix Pipeline Support**: Pipe paths in and results out - works seamlessly in command chains
- **Configurable**: YAML configuration file with sensible defaults
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Fuzzy Finder Integration**: Perfect for fuzzy finding and quick navigation with `fzf`, `television`, or your favorite fuzzy finder

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

#### Debian/Ubuntu (.deb)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_amd64.deb
sudo dpkg -i pj_0.1.0_linux_amd64.deb

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_arm64.deb
sudo dpkg -i pj_0.1.0_linux_arm64.deb
```

#### Fedora/RHEL/CentOS (.rpm)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_amd64.rpm
sudo rpm -i pj_0.1.0_linux_amd64.rpm

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_arm64.rpm
sudo rpm -i pj_0.1.0_linux_arm64.rpm
```

#### Alpine (.apk)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_amd64.apk
apk add --allow-untrusted pj_0.1.0_linux_amd64.apk

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_arm64.apk
apk add --allow-untrusted pj_0.1.0_linux_arm64.apk
```

#### Arch Linux (.pkg.tar.zst)

```bash
# amd64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_amd64.pkg.tar.zst
sudo pacman -U pj_0.1.0_linux_amd64.pkg.tar.zst

# arm64
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_linux_arm64.pkg.tar.zst
sudo pacman -U pj_0.1.0_linux_arm64.pkg.tar.zst
```

### Pre-built Binaries (All Platforms)

Download the latest binaries for your platform from the [releases page](https://github.com/josephschmitt/pj/releases).

Available platforms:
- **macOS**: `darwin_amd64` (Intel), `darwin_arm64` (Apple Silicon)
- **Linux**: `linux_amd64`, `linux_arm64`
- **Windows**: `windows_amd64`, `windows_arm64`

```bash
# Example: macOS Apple Silicon
wget https://github.com/josephschmitt/pj/releases/download/v0.1.0/pj_0.1.0_darwin_arm64.tar.gz
tar -xzf pj_0.1.0_darwin_arm64.tar.gz
sudo mv pj /usr/local/bin/
```

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
| `--strip` | | Strip icons from output |
| `--icon-map MARKER:ICON` | | Override icon mapping |
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

# Files/directories that mark a project
markers:
  - .git
  - go.mod
  - package.json
  - Cargo.toml
  - pyproject.toml
  - Makefile
  - flake.nix
  - composer.json
  - build.gradle

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

# Icon mappings (requires Nerd Fonts)
icons:
  .git: ""
  go.mod: "󰟓"
  package.json: "󰎙"
  Cargo.toml: ""
  pyproject.toml: ""
  Makefile: ""
  flake.nix: ""
  composer.json: ""
```

### Config Priority

CLI flags override config file settings, which override defaults.

```bash
# Config says max_depth: 3, this overrides to 5
pj -d 5
```

## Integration with Fuzzy Finders

The real power of `pj` comes from integrating it with fuzzy finders like `fzf` or `television` for quick project navigation.

### Shell Function (Bash/Zsh)

Add to your `~/.bashrc` or `~/.zshrc`:

#### Using fzf

```bash
# Quick project navigation with fzf
pjf() {
  local project
  project=$(pj --icons | fzf --ansi --preview 'ls -la {2}' --preview-window right:60%) &&
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
  project=$(pj --icons | tv) &&
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
    set -l project (pj --icons | fzf --ansi --preview 'ls -la (echo {} | awk \'{print $2}\')' --preview-window right:60%)
    and cd (echo $project | awk '{print $2}')
end
```

#### Using television

```fish
function pjt
    set -l project (pj --icons | tv)
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

### Vim/Neovim Integration

Use with telescope or fzf.vim to quickly open projects in your editor.

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

- [fzf](https://github.com/junegunn/fzf) - Command-line fuzzy finder
- [television](https://github.com/alexpasmantier/television) - Blazingly fast fuzzy finder TUI
- [gah](https://github.com/get-gah/gah) - GitHub Asset Helper for installing binaries
- [z](https://github.com/rupa/z) - Jump around directories
- [autojump](https://github.com/wting/autojump) - Fast way to navigate filesystem

---

**Note**: Icon display requires a [Nerd Font](https://www.nerdfonts.com/) to be installed and used in your terminal.
