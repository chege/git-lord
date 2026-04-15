# Contributing to git-lord

Thank you for your interest in contributing to git-lord! This document provides guidelines and instructions for contributing.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/chege/git-lord.git
cd git-lord

# Install dependencies and build
go mod download
make build

# Run tests
make test
```

## Commit Message Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/) specification. All commit messages and pull request titles must follow this format:

```
<type>[(optional scope)]: <description>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (formatting, semicolons, etc.)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding or correcting tests
- **build**: Changes to build system or dependencies
- **ci**: Changes to CI configuration
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Examples

```
feat: add support for CSV output format

fix(parser): handle empty commit messages correctly

docs: update installation instructions for macOS

feat(cli)!: remove deprecated --json flag

BREAKING CHANGE: The --json flag has been removed. Use --format json instead.
```

### Scopes

Common scopes for this project:
- `core`: Core functionality and business logic
- `cli`: Command-line interface and flags
- `deps`: Dependency updates
- `release`: Release-related changes
- `brew`: Homebrew formula updates

## Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** with appropriate tests
3. **Ensure tests pass**: `make test`
4. **Ensure linting passes**: `make lint`
5. **Commit with conventional commits** format
6. **Push your branch** and create a pull request

## Release Process

This project uses automated releases based on conventional commits:

- `fix:` commits trigger patch releases (0.0.x)
- `feat:` commits trigger minor releases (0.x.0)
- Commits with `BREAKING CHANGE:` trigger major releases (x.0.0)

Releases are published automatically to:
- GitHub Releases
- Homebrew tap (chege/homebrew-tap)

## Homebrew Tap

Users can install git-lord via Homebrew:

```bash
brew tap chege/tap
brew install git-lord
```

Or install directly:

```bash
brew install chege/tap/git-lord
```

The Homebrew formula is automatically updated when new releases are published.

## Code Style

- Follow standard Go conventions (`gofmt`)
- Run `make format` before committing
- Keep functions focused and well-documented
- Add tests for new functionality

## Testing

```bash
# Run all tests
make test

# Run with race detector
go test -race ./...

# Run specific test
go test -v ./internal/... -run TestFunctionName
```

## Questions?

Feel free to open an issue for questions or discussion.
