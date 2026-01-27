# bak

A simple CLI wrapper for [restic](https://restic.net/) backups, designed for homelab environments.

## Overview

`bak` simplifies restic backup configuration and management by providing:

- Easy setup with sensible defaults
- Automatic systemd timer creation for scheduled backups
- Server-side retention via tags (compatible with append-only servers)
- Simple commands for common operations

## Prerequisites

### Required: restic

`bak` is a wrapper around [restic](https://restic.net/) and requires it to be installed and available in your PATH.

**Install restic:**

```bash
# Debian/Ubuntu
sudo apt install restic

# Fedora/RHEL
sudo dnf install restic

# Arch Linux
sudo pacman -S restic

# macOS (Homebrew)
brew install restic

# Or download from https://github.com/restic/restic/releases
```

Verify installation:

```bash
restic version
```

### Optional: systemd

systemd is required for scheduled automatic backups. Manual backups with `bak now` work without systemd.

## Installation

### From Source

```bash
git clone https://github.com/magicmicky/bak.git
cd bak
make build
sudo make install
```

### Manual

```bash
go build -o bak ./cmd/bak
sudo install -m 755 bak /usr/local/bin/bak
```

## Usage

### Initialize Credentials

First, configure repository credentials (one-time per machine):

```bash
# Interactive mode
sudo bak init --repo rest:https://user@backup.example.com:8000

# Non-interactive mode
sudo bak init --repo rest:https://user@backup.example.com:8000 --password "secret"

# Using environment variables
sudo RESTIC_REPOSITORY=rest:https://... RESTIC_PASSWORD=secret bak init

# Preview changes without writing
sudo bak init --repo ... --password ... --dry-run
```

### Configure Backups

After credentials are set up, configure automated backups:

```bash
sudo bak setup --tag webapp --paths /var/www,/etc/nginx
```

With custom retention and schedule:

```bash
sudo bak setup --tag gameserver --paths /opt/game/saves \
    --schedule hourly \
    --keep-hourly 24 \
    --keep-daily 7
```

### Run Backup Now

Execute a backup immediately with live output:

```bash
sudo bak now
```

### Check Status

View configuration and recent snapshots:

```bash
bak status
```

### Edit Configuration

Modify existing configuration:

```bash
sudo bak edit --keep-daily 14 --keep-weekly 8
sudo bak edit --paths /var/www,/etc/nginx,/opt/certs
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize repository credentials (one-time per machine) |
| `setup` | Configure automated backups (creates config + systemd timer) |
| `now` | Run backup immediately |
| `status` | Show configuration and recent snapshots |
| `edit` | Modify existing configuration |
| `list` | List snapshots with detailed information |
| `logs` | Show recent backup logs from journald |
| `completion` | Generate shell completion scripts (bash, zsh, fish, powershell) |

## Init Options

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `$RESTIC_REPOSITORY` | Repository URL (prompted if not provided) |
| `--password` | `$RESTIC_PASSWORD` | Repository password (prompted if not provided) |
| `--force` | `false` | Overwrite existing credentials |
| `--dry-run` | `false` | Preview changes without writing |

## Setup Options

| Flag | Default | Description |
|------|---------|-------------|
| `--tag` | (required) | Unique identifier for this host's backups |
| `--paths` | (required) | Comma-separated directories to backup |
| `--schedule` | `daily` | Schedule: `daily`, `hourly`, or cron expression |
| `--keep-hourly` | `0` | Hourly snapshots to keep |
| `--keep-daily` | `7` | Daily snapshots to keep |
| `--keep-weekly` | `4` | Weekly snapshots to keep |
| `--keep-monthly` | `6` | Monthly snapshots to keep |
| `--keep-yearly` | `0` | Yearly snapshots to keep |
| `--exclude` | | Exclude pattern (repeatable) |
| `--notify` | | Apprise notification URL |
| `--force` | `false` | Overwrite existing configuration |
| `--dry-run` | `false` | Preview changes without writing |

## Edit Options

| Flag | Default | Description |
|------|---------|-------------|
| `--paths` | | Update backup paths |
| `--schedule` | | Update schedule |
| `--keep-hourly` | | Update hourly retention |
| `--keep-daily` | | Update daily retention |
| `--keep-weekly` | | Update weekly retention |
| `--keep-monthly` | | Update monthly retention |
| `--keep-yearly` | | Update yearly retention |
| `--exclude` | | Update exclude patterns |
| `--notify` | | Update notification URL |
| `--yes` | `false` | Skip confirmation prompt |
| `--dry-run` | `false` | Preview changes without writing |

## List Options

| Flag | Default | Description |
|------|---------|-------------|
| `-n`, `--last` | `10` | Number of snapshots to show |

## Logs Options

| Flag | Default | Description |
|------|---------|-------------|
| `-n`, `--lines` | `20` | Number of log lines to show |

## Shell Completion

Generate and load shell completions:

```bash
# Bash (add to ~/.bashrc)
eval "$(bak completion bash)"

# Zsh (add to ~/.zshrc)
eval "$(bak completion zsh)"

# Fish
bak completion fish > ~/.config/fish/completions/bak.fish
```

## Configuration Files

### `/etc/backup/env`

Environment variables for restic (created by `bak init`):

```bash
RESTIC_REPOSITORY="rest:https://user@backups.example.com/"
RESTIC_PASSWORD_FILE="/etc/backup/restic-password"
RESTIC_CACHE_DIR="/var/cache/restic"
```

### `/etc/backup/restic-password`

Repository encryption password (created by `bak init`, mode 0600).

### `/etc/backup/backup.conf`

Per-host backup configuration (managed by `bak`):

```bash
BACKUP_TAG="myapp"
BACKUP_PATHS="/var/www,/etc/nginx"
BACKUP_EXCLUDES="*.log node_modules"
KEEP_HOURLY=0
KEEP_DAILY=7
KEEP_WEEKLY=4
KEEP_MONTHLY=6
KEEP_YEARLY=0
NOTIFY_URL=""
```

## Architecture

This tool is designed for use with:

- **Append-only backup server**: Clients cannot delete snapshots
- **Server-side retention**: Retention is declared via tags (e.g., `retain:h=24,d=7,w=4,m=6`) and enforced by server-side processes
- **Credential management**: Use `bak init` to configure credentials, or pre-provision `/etc/backup/` for automated deployments

## License

MIT License - see [LICENSE](LICENSE) for details.
