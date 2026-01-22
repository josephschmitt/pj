<div align="center">
  <img src="docs/image.png" alt="pj logo" width="600">

  # pj - Project Finder CLI

  Fast project directory finder that searches your filesystem for git repositories and project directories. Built for speed and seamless integration with fuzzy finders like [fzf](https://github.com/junegunn/fzf) and [television](https://github.com/alexpasmantier/television).
</div>

## Features

- **Fast Discovery**: Quickly finds all projects across multiple search paths
- **Smart Caching**: Caches results for instant subsequent searches (5-minute TTL)
- **Flexible Markers**: Detects projects by `.git`, `go.mod`, `package.json`, `Cargo.toml`, and more
- **Icon Support**: Display pretty icons for different project types (Nerd Fonts required)
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
gah install pj
```

Note: After the first release, a PR will be submitted to add `pj` to the gah database. Until then, use `gah install josephschmitt/pj`.

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

## Development

### Building

```bash
# Build for current platform
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint

# Build for all platforms
make build-all
```

### Testing GoReleaser Locally

```bash
# Validate config
make release-check

# Test release build (without publishing)
make release-local
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes using [Conventional Commits](#conventional-commits)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/) to automate releases and changelog generation. Please format your commit messages as:

```
<type>: <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` - New feature (triggers minor version bump)
- `fix:` - Bug fix (triggers patch version bump)
- `docs:` - Documentation only changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks
- `ci:` - CI/CD changes

**Breaking Changes:** Add `!` after type or include `BREAKING CHANGE:` in footer to trigger major version bump.

**Examples:**
```bash
feat: add support for custom config directory
fix: resolve cache invalidation issue
docs: update installation instructions
feat!: change default search depth to 5
```

### Release Process

Releases are fully automated:

1. When commits are pushed to `main`, [release-please](https://github.com/googleapis/release-please) analyzes commit messages
2. A release PR is automatically created/updated with changelog and version bump
3. When the release PR is merged, a git tag is created
4. The tag triggers [GoReleaser](https://goreleaser.com/) to build and publish:
   - GitHub release with binaries
   - Homebrew formula
   - Scoop manifest
   - Linux packages (deb, rpm, apk, Arch)

No manual tagging or version bumping required!

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
