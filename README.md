# bak

A simple CLI wrapper for [restic](https://restic.net/) backups, designed for homelab environments.

## Overview

`bak` simplifies restic backup configuration and management by providing:

- Easy setup with sensible defaults
- Automatic systemd timer creation for scheduled backups
- Server-side retention via tags (compatible with append-only servers)
- Simple commands for common operations

## Prerequisites

- restic installed and available in PATH
- Credentials pre-configured in `/etc/backup/`:
  - `env` - Environment variables (RESTIC_REPOSITORY, RESTIC_PASSWORD_FILE, etc.)
  - `restic-password` - Repository encryption password
- systemd (for scheduled backups)

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

### Initial Setup

Configure automated backups for this host:

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
| `setup` | Configure automated backups (creates config + systemd timer) |
| `now` | Run backup immediately |
| `status` | Show configuration and recent snapshots |
| `edit` | Modify existing configuration |

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

## Configuration Files

### `/etc/backup/env`

Environment variables for restic:

```bash
RESTIC_REPOSITORY="rest:https://user:pass@backups.example.com/"
RESTIC_PASSWORD_FILE="/etc/backup/restic-password"
RESTIC_CACHE_DIR="/var/cache/restic"
```

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
- **Pre-configured credentials**: VMs ship with credentials already in `/etc/backup/`

## Development

### Running Tests

```bash
# Run unit tests
make test

# Run all tests (unit + integration)
make test-all
```

### Integration Tests

Integration tests run in a Docker container with systemd and restic to test the full CLI workflow.

```bash
# Run integration tests (automated)
make test-integration
```

### Manual Testing with Docker

For interactive testing and debugging, you can start the test environment and shell into it:

```bash
# Build and start the test container
make test-integration-up

# Shell into the running container
make test-integration-shell
```

Inside the container, you have access to:
- `bak` binary at `/usr/local/bin/bak`
- `restic` for direct repository inspection
- Pre-initialized restic repository at `/tmp/restic-repo`
- Sample test data at `/tmp/backup-source`
- Environment variables `RESTIC_REPOSITORY` and `RESTIC_PASSWORD` already configured

Example manual test session:

```bash
# Inside the container
bak setup --tag manual-test --paths /tmp/backup-source
bak now
bak status
restic snapshots

# Run specific integration tests
go test -v -tags=integration ./test/integration/... -run TestSetup_BasicConfiguration
```

When done:

```bash
# Clean up
make test-integration-down
```

Other useful commands:

```bash
# View test service logs (when running automated tests)
make test-integration-logs

# Clean up Docker images
make test-integration-clean
```

## License

MIT License - see [LICENSE](LICENSE) for details.
