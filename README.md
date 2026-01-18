# Gurren

SSH tunnel manager for connecting to remote services through bastion hosts.

## Installation

```bash
go install github.com/JoshElias/gurren@latest
```

Or build from source:

```bash
git clone https://github.com/JoshElias/gurren.git
cd gurren
go build .
```

## Usage

### Using a named tunnel from config

```bash
gurren connect production-db
```

### Using flags directly

```bash
gurren connect --host ec2-user@bastion.example.com --remote db.internal:3306 --local 127.0.0.1:3306
```

### Override auth method

```bash
gurren connect production-db --auth publickey
```

## Configuration

Gurren looks for config files in this order:

1. `~/.config/gurren/config.toml`
2. `~/gurren.toml`

### Example config

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
| `publickey` | Private key file (`~/.ssh/id_ed25519`, `~/.ssh/id_ecdsa`, `~/.ssh/id_rsa`) | 2 |
| `password` | Password prompt | 3 (last resort) |

When `method = "auto"` (default), Gurren tries each method in priority order until one succeeds. Methods fail silently except for password, which prompts for input.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Path to config file |
| `--auth` | `-a` | Auth method: `auto`, `agent`, `publickey`, `password` |
| `--host` | | SSH host (`user@host:port`) |
| `--remote` | | Remote address (`host:port`) |
| `--local` | | Local bind address (`host:port`) |
