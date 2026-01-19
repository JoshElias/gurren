# Gurren

A fast, terminal-based SSH tunnel manager with a TUI and background service.

<img width="893" height="550" alt="image" src="https://github.com/user-attachments/assets/843c0e06-138b-4e59-ad60-c5be25d5a500" />


## Features

- **Interactive TUI** — Manage all your tunnels from a single interface
- **Background service** — Tunnels persist even after closing the TUI
- **systemd integration** — Optional systemd user service for auto-start on login
- **Vim-style navigation** — `j`/`k` to navigate, `Enter` to toggle
- **Multiple auth methods** — SSH agent, public key, and password
- **SSH config support** — Use hosts from `~/.ssh/config` directly
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

Launches the interactive terminal interface. The service starts automatically if not running.

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

# Direct connection with flags (bypasses config)
# --host accepts user@host:port or a Host from ~/.ssh/config
gurren connect --host user@bastion:22 --remote db:5432 --local localhost:5432
gurren connect --host my-ssh-host --remote db:5432 --local localhost:5432

# Service management
gurren service start    # Start service in background
gurren service stop     # Stop service and all tunnels
gurren service status   # Check if service is running

# systemd integration (Linux only)
gurren service install    # Install systemd user service
gurren service uninstall  # Remove systemd user service
gurren service enable     # Enable auto-start on login
gurren service disable    # Disable auto-start

# Shell completion
gurren completion bash       # Bash completion script
gurren completion zsh        # Zsh completion script
gurren completion fish       # Fish completion script
gurren completion powershell # PowerShell completion script
```

### Global Flags

These flags work with any command:

| Flag | Description |
|------|-------------|
| `--config <path>` | Config file path (default: `~/.config/gurren/config.toml`) |
| `-a, --auth <method>` | Auth method: `auto`, `agent`, `publickey`, `password` (default: `auto`) |

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

[[tunnels]]
# name is optional - derived from host if omitted (becomes "bastion")
host = "bastion"
remote = "redis.internal:6379"
local = "localhost:6379"
```

## Authentication

Gurren supports three SSH authentication methods:

| Method | Description | Priority |
|--------|-------------|----------|
| `agent` | SSH agent (uses `SSH_AUTH_SOCK`) | 1 (tried first) |
| `publickey` | Private key files (`~/.ssh/id_ed25519`, `id_ecdsa`, `id_rsa`) | 2 |
| `password` | Interactive password prompt | 3 (last resort) |

When `method = "auto"` (default), Gurren tries each method in priority order until one succeeds.

## SSH Config Integration

Gurren reads your `~/.ssh/config` file and can use any `Host` entry directly. This means you can reference hosts by their alias instead of specifying full connection details.

For example, if your SSH config contains:

```
Host bastion
    HostName bastion.example.com
    User ec2-user
    Port 22
```

You can connect using just the alias:

```bash
gurren connect --host bastion --remote db.internal:5432 --local localhost:5432
```

In your `config.toml`, you can also reference SSH config hosts:

```toml
[[tunnels]]
name = "production-db"
host = "bastion"  # Uses Host entry from ~/.ssh/config
remote = "db.internal:5432"
local = "localhost:5432"
```

If you omit the `name` field, Gurren will automatically use the host value as the tunnel name (stripping the `user@` prefix if present). Duplicate names are auto-suffixed (e.g., `bastion`, `bastion-2`).

## systemd Integration

On Linux systems with systemd, you can install Gurren as a user service for automatic startup on login.

### Install and Enable

```bash
# Install the systemd user service
gurren service install

# Enable auto-start on login
gurren service enable

# Start the service now (or just reboot/re-login)
systemctl --user start gurren
```

### Manage with systemctl

Once installed, you can also manage the service directly with systemctl:

```bash
# Check status
systemctl --user status gurren

# View logs
journalctl --user -u gurren

# Restart service
systemctl --user restart gurren
```

### Uninstall

```bash
gurren service uninstall
```

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
         │   Service             │
         │   - Manages tunnels   │
         │   - Tracks state      │
         │   - Pushes updates    │
         └───────────────────────┘
```

- **Service** runs in the background and manages SSH tunnel lifecycles
- **TUI** and **CLI** are clients that communicate with the service via Unix socket
- **Tunnels persist** after the TUI exits — the service keeps them running
- **Status updates** are pushed from service to subscribed clients in real-time

## Roadmap

- [ ] Homebrew formula
- [x] AUR package
- [x] SSH config file (`~/.ssh/config`) parsing
- [x] systemd user service support
- [ ] Host key verification
- [ ] Test coverage

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to help with these.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
