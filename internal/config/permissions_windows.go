//go:build windows

package config

import (
	"fmt"
	"os/exec"
)

// restrictPermissions uses icacls to remove inherited permissions and grant
// access only to SYSTEM and Administrators, preventing other users from
// reading sensitive files like the restic password.
func restrictPermissions(path string) error {
	cmd := exec.Command("icacls", path,
		"/inheritance:r",
		"/grant:r", "SYSTEM:(R)",
		"/grant:r", "Administrators:(F)",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w\n%s", path, err, out)
	}
	return nil
}
