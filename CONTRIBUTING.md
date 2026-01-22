# Contributing to pj

Thank you for your interest in contributing to `pj`! This guide will help you get started with development and submitting contributions.

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, but recommended)

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

## Conventional Commits

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

## Release Process

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

## Testing Guidelines

All new features and bug fixes should include tests. See [CLAUDE.md](CLAUDE.md) for detailed testing patterns and coverage requirements.

## Code Quality

- Follow Go best practices and idioms
- Keep functions focused and concise
- Avoid unnecessary comments (see [CLAUDE.md](CLAUDE.md) for commenting guidelines)
- Run `make lint` before submitting PRs
