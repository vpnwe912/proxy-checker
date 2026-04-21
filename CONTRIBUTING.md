# Contributing to Proxy Checker

First off, thank you for considering contributing to Proxy Checker! 🎉

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- Use a clear and descriptive title
- Describe the exact steps to reproduce the problem
- Provide specific examples
- Describe the behavior you observed and what you expected
- Include screenshots if possible
- Mention your OS version and Proxy Checker version

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- Use a clear and descriptive title
- Provide a detailed description of the suggested enhancement
- Explain why this enhancement would be useful
- List some examples of how it would be used

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code, add tests
3. Make sure your code follows the existing style
4. Write a clear commit message

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- Wails CLI v2
- Make (optional, for Windows: `winget install GnuWin32.Make`)

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/proxy-checker.git
cd proxy-checker

# Install Go dependencies
go mod download

# Build
make build

# Or run in dev mode
make dev
```

## Project Structure

```
proxy-checker/
├── main.go              # Application entry point
├── wails_main.go        # Wails desktop app
├── proxy.go             # Proxy parsing and checking
├── threeproxy.go        # 3proxy management
├── version.go           # Version management
├── frontend/            # Web UI (HTML/CSS/JS)
├── embedded/3proxy/     # Embedded 3proxy binaries
└── scripts/             # Build scripts
```

## Coding Guidelines

### Go Code

- Follow standard Go formatting (`gofmt`)
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused
- Handle errors properly

### JavaScript Code

- Use modern ES6+ syntax
- Keep functions pure when possible
- Use meaningful variable names
- Add comments for complex UI logic

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Use imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit first line to 72 characters
- Reference issues and pull requests

Examples:
```
feat: add support for SOCKS4 proxies
fix: resolve memory leak in monitor
docs: update README with new features
chore: bump version to 1.2.0
```

## Testing

```bash
# Run unit tests
go test ./...

# Run specific test
go test -run TestProxyParsing
```

## Building

```bash
# Development build
make dev

# Production build
make build

# Clean build
make rebuild
```

## Version Management

Version is managed automatically:
- Each `make build` increments patch version (1.0.0 → 1.0.1)
- When patch reaches 10, minor increments (1.0.9 → 1.1.0)
- Edit `wails.json` to set version manually

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

Thank you for contributing! 🚀
