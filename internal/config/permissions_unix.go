//go:build !windows

package config

// restrictPermissions is a no-op on Unix; file mode 0600 is set at write time.
func restrictPermissions(path string) error {
	return nil
}
