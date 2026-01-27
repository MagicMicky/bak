//go:build integration

package integration

import (
	"testing"
)

func TestStatus_ShowsConfiguration(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-status",
		"--paths", backupSource,
		"--schedule", "hourly",
		"--keep-daily", "14",
		"--keep-weekly", "8",
	)

	// Run status
	stdout, _ := env.RunBakExpectSuccess("status")

	// Verify configuration is shown
	env.AssertOutputContains(stdout, "=== Configuration ===")
	env.AssertOutputContains(stdout, "Tag:      test-status")
	env.AssertOutputContains(stdout, backupSource)
	env.AssertOutputContains(stdout, "Schedule: hourly")
	env.AssertOutputContains(stdout, "daily=14")
	env.AssertOutputContains(stdout, "weekly=8")
}

func TestStatus_ShowsTimerStatus(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-timer-status",
		"--paths", backupSource,
	)

	// Run status
	stdout, _ := env.RunBakExpectSuccess("status")

	// Verify timer status is shown
	env.AssertOutputContains(stdout, "=== Timer Status ===")
	// Timer should show some status information
	env.AssertOutputContains(stdout, "backup.timer")
}

func TestStatus_ShowsSnapshots(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-status-snaps",
		"--paths", backupSource,
	)

	// Run a backup to create a snapshot
	env.RunBakExpectSuccess("now")

	// Run status
	stdout, _ := env.RunBakExpectSuccess("status")

	// Verify snapshots section is shown
	env.AssertOutputContains(stdout, "=== Recent Snapshots ===")
	// After backup, should show snapshot info (short ID appears in output)
	env.AssertOutputContains(stdout, "test-status-snaps")
}

func TestStatus_FailsWithoutSetup(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run status without setup
	_, stderr, _ := env.RunBakExpectError("status")
	env.AssertOutputContains(stderr, "not configured")
	env.AssertOutputContains(stderr, "Run 'bak setup' first")
}

func TestStatus_ShowsExcludes(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup with excludes
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-status-excludes",
		"--paths", backupSource,
		"--exclude", "*.log",
		"--exclude", "*.tmp",
	)

	// Run status
	stdout, _ := env.RunBakExpectSuccess("status")

	// Verify excludes are shown
	env.AssertOutputContains(stdout, "Excludes:")
	env.AssertOutputContains(stdout, "*.log")
	env.AssertOutputContains(stdout, "*.tmp")
}

func TestStatus_NoSnapshotsYet(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup without running backup
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-no-snaps",
		"--paths", backupSource,
	)

	// Run status - should not error even without snapshots
	stdout, _ := env.RunBakExpectSuccess("status")

	// Should still show configuration
	env.AssertOutputContains(stdout, "=== Configuration ===")
	env.AssertOutputContains(stdout, "Tag:      test-no-snaps")
}
