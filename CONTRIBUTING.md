# Contributing to Gurren

Thanks for your interest in contributing to Gurren! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git

### Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gurren.git
   cd gurren
   ```
3. Build the project:
   ```bash
   go build .
   ```
4. Run it:
   ```bash
   ./gurren
   ```

## Project Structure

```
gurren/
├── main.go                 # Entry point
├── internal/
│   ├── cmd/                # CLI commands (Cobra)
│   ├── config/             # Configuration loading (Viper)
│   ├── auth/               # SSH authentication methods
│   ├── tunnel/             # SSH tunnel management
│   ├── daemon/             # Background service (IPC server)
│   └── tui/                # Terminal UI (BubbleTea)
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `cmd` | CLI commands and flags |
| `config` | TOML config parsing |
| `auth` | SSH auth (agent, publickey, password) |
| `tunnel` | SSH tunnel lifecycle, state management |
| `daemon` | Background service, Unix socket IPC |
| `tui` | BubbleTea-based terminal interface |

## Development

### Running Locally

```bash
# Build and run TUI
go build . && ./gurren

# Run service in foreground (for debugging)
./gurren service start --foreground

# In another terminal
./gurren ls
./gurren connect my-tunnel
```

### Running Tests

```bash
go test ./...
```

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions focused and small
- Add comments for exported functions and types
- Error messages should be lowercase and not end with punctuation

## Pull Requests

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. Make your changes with clear, focused commits

3. Ensure the project builds and tests pass:
   ```bash
   go build ./...
   go test ./...
   ```

4. Push to your fork and open a Pull Request

### PR Guidelines

- Keep PRs focused on a single change
- Update documentation if needed
- Add tests for new functionality
- Follow existing code patterns

## Reporting Issues

### Bug Reports

Please include:
- Gurren version (`gurren --version` or commit hash)
- Operating system and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant config (redact sensitive info)

### Feature Requests

- Describe the use case
- Explain why existing functionality doesn't meet the need
- If possible, suggest an approach

## Areas for Contribution

Check the [Roadmap](README.md#roadmap) for planned features. Some good first issues:

- Improving test coverage
- Documentation improvements
- Bug fixes
- SSH config file parsing support

## Questions?

Open an issue with the `question` label or start a discussion.
