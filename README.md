# Gurren

A fast, terminal-based SSH tunnel manager with a TUI and background daemon.

<!-- TODO: Add screenshot/GIF here -->
<!-- ![Gurren TUI](./assets/screenshot.png) -->

## Features

- **Interactive TUI** — Manage all your tunnels from a single interface
- **Background daemon** — Tunnels persist even after closing the TUI
- **Vim-style navigation** — `j`/`k` to navigate, `Enter` to toggle
- **Multiple auth methods** — SSH agent, public key, and password
- **Simple configuration** — TOML-based config file
- **Real-time status** — Push-based status updates in the TUI

## Installation

### From Source

```bash
go install github.com/JoshElias/gurren@latest
```

Or build manually:

```bash
git clone https://github.com/JoshElias/gurren.git
cd gurren
go build .
```

### Homebrew

Coming soon.

### AUR

```bash
# Stable release
yay -S gurren

# Development version (latest git)
yay -S gurren-git
```

## Quick Start

1. Create a config file at `~/.config/gurren/config.toml`:

```toml
[[tunnels]]
name = "my-database"
host = "user@bastion.example.com"
remote = "db.internal:5432"
local = "localhost:5432"
```

2. Launch the TUI:

```bash
gurren
```

3. Use `j`/`k` to navigate, `Enter` to connect/disconnect, `q` to quit.

## Usage

### TUI (Default)

```bash
gurren
```

Launches the interactive terminal interface. The daemon starts automatically if not running.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Toggle connection |
| `q` | Quit (tunnels keep running) |

### CLI Commands

```bash
# List all tunnels with status
gurren ls
gurren ls --json

# Connect/disconnect via CLI
gurren connect my-database
gurren disconnect my-database

# Direct connection with flags (bypasses daemon)
gurren connect --host user@bastion:22 --remote db:5432 --local localhost:5432

# Daemon management
gurren daemon start    # Start daemon (foreground)
gurren daemon stop     # Stop daemon and all tunnels
gurren daemon status   # Check if daemon is running
```

## Configuration

Gurren looks for config files in this order:

1. `~/.config/gurren/config.toml`
2. `~/gurren.toml`

### Example Config

```toml
[auth]
method = "auto"  # "auto", "agent", "publickey", or "password"

[[tunnels]]
name = "production-db"
host = "ec2-user@bastion.example.com"
remote = "db.internal:3306"
local = "127.0.0.1:3306"

[[tunnels]]
name = "staging-db"
host = "ec2-user@bastion-staging.example.com"
remote = "db-staging.internal:3306"
local = "127.0.0.1:3307"
```

## Authentication

Gurren supports three SSH authentication methods:

| Method | Description | Priority |
|--------|-------------|----------|
| `agent` | SSH agent (uses `SSH_AUTH_SOCK`) | 1 (tried first) |
| `publickey` | Private key files (`~/.ssh/id_ed25519`, `id_ecdsa`, `id_rsa`) | 2 |
| `password` | Interactive password prompt | 3 (last resort) |

When `method = "auto"` (default), Gurren tries each method in priority order until one succeeds.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   gurren (TUI)  │     │   gurren ls     │
│                 │     │   gurren connect│
└────────┬────────┘     └────────┬────────┘
         │   Unix socket (JSON)  │
         └───────────┬───────────┘
                     ▼
         ┌───────────────────────┐
         │   Daemon              │
         │   - Manages tunnels   │
         │   - Tracks state      │
         │   - Pushes updates    │
         └───────────────────────┘
```

- **Daemon** runs in the background and manages SSH tunnel lifecycles
- **TUI** and **CLI** are clients that communicate with the daemon via Unix socket
- **Tunnels persist** after the TUI exits — the daemon keeps them running
- **Status updates** are pushed from daemon to subscribed clients in real-time

## Roadmap

- [ ] Homebrew formula
- [x] AUR package
- [x] SSH config file (`~/.ssh/config`) parsing
- [ ] Host key verification
- [ ] Test coverage

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to help with these.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
