//go:build windows

package config

import (
	"os"
	"path/filepath"
)

var (
	// DefaultConfigDir is the default directory for backup configuration files.
	DefaultConfigDir = configDir()
	// DefaultConfigPath is the default location for the backup configuration.
	DefaultConfigPath = filepath.Join(configDir(), "backup.conf")
	// DefaultEnvPath is the default location for the restic environment file.
	DefaultEnvPath = filepath.Join(configDir(), "env")
	// DefaultPasswordPath is the default location for the restic password file.
	DefaultPasswordPath = filepath.Join(configDir(), "restic-password")
	// DefaultCacheDir is the default location for the restic cache.
	DefaultCacheDir = filepath.Join(configDir(), "cache")
)

func configDir() string {
	if d := os.Getenv("PROGRAMDATA"); d != "" {
		return filepath.Join(d, "bak")
	}
	return filepath.Join("C:\\ProgramData", "bak")
}
