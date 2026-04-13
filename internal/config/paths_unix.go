//go:build !windows

package config

var (
	// DefaultConfigDir is the default directory for backup configuration files.
	DefaultConfigDir = "/etc/backup"
	// DefaultConfigPath is the default location for the backup configuration.
	DefaultConfigPath = "/etc/backup/backup.conf"
	// DefaultEnvPath is the default location for the restic environment file.
	DefaultEnvPath = "/etc/backup/env"
	// DefaultPasswordPath is the default location for the restic password file.
	DefaultPasswordPath = "/etc/backup/restic-password"
	// DefaultCacheDir is the default location for the restic cache.
	DefaultCacheDir = "/var/cache/restic"
)
