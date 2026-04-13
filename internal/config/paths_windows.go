//go:build windows

package config

import (
	"os"
	"path/filepath"
)

var cfgDir = configDir()

var (
	// DefaultConfigDir is the default directory for backup configuration files.
	DefaultConfigDir = cfgDir
	// DefaultConfigPath is the default location for the backup configuration.
	DefaultConfigPath = filepath.Join(cfgDir, "backup.conf")
	// DefaultEnvPath is the default location for the restic environment file.
	DefaultEnvPath = filepath.Join(cfgDir, "env")
	// DefaultPasswordPath is the default location for the restic password file.
	DefaultPasswordPath = filepath.Join(cfgDir, "restic-password")
	// DefaultCacheDir is the default location for the restic cache.
	DefaultCacheDir = filepath.Join(cfgDir, "cache")
)

func configDir() string {
	if d := os.Getenv("PROGRAMDATA"); d != "" {
		return filepath.Join(d, "bak")
	}
	return filepath.Join("C:\\ProgramData", "bak")
}
