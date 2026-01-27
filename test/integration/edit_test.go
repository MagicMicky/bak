//go:build integration

package integration

import (
	"testing"
)

func TestEdit_UpdatesPaths(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-paths",
		"--paths", backupSource,
	)

	// Create new directory for testing path change
	newPath := "/tmp/backup-source-new"
	env.RunBak("mkdir", "-p", newPath) // Note: using system mkdir, not bak

	// Edit paths (use --yes to skip confirmation)
	stdout, _ := env.RunBakExpectSuccess(
		"edit",
		"--paths", newPath,
		"--yes",
	)

	// Verify output shows the change
	env.AssertOutputContains(stdout, "Proposed changes:")
	env.AssertOutputContains(stdout, "Configuration updated successfully")

	// Verify config file was updated
	env.AssertFileContains(configPath, `BACKUP_PATHS="`+newPath+`"`)
}

func TestEdit_UpdatesScheduleRestartsTimer(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup with daily schedule
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-schedule",
		"--paths", backupSource,
		"--schedule", "daily",
	)

	// Remember original timer content
	originalTimer := env.ReadFile(timerPath)

	// Edit schedule to hourly
	stdout, _ := env.RunBakExpectSuccess(
		"edit",
		"--schedule", "hourly",
		"--yes",
	)

	// Verify output
	env.AssertOutputContains(stdout, "schedule: daily -> hourly")
	env.AssertOutputContains(stdout, "Timer updated and restarted")

	// Verify config was updated
	env.AssertFileContains(configPath, `BACKUP_SCHEDULE="hourly"`)

	// Verify timer file changed
	newTimer := env.ReadFile(timerPath)
	if originalTimer == newTimer {
		t.Error("timer file should have changed when schedule was updated")
	}

	// Timer should still be active
	if !env.TimerIsActive() {
		t.Error("timer should be active after schedule change")
	}
}

func TestEdit_NoChanges(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-nochange",
		"--paths", backupSource,
	)

	// Edit without specifying any flags
	stdout, _ := env.RunBakExpectSuccess("edit")

	// Verify "No changes" message
	env.AssertOutputContains(stdout, "No changes specified")
}

func TestEdit_UpdatesRetention(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup with default retention
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-retention",
		"--paths", backupSource,
	)

	// Verify initial retention
	env.AssertFileContains(configPath, "KEEP_DAILY=7") // default

	// Edit retention values
	stdout, _ := env.RunBakExpectSuccess(
		"edit",
		"--keep-daily", "30",
		"--keep-weekly", "12",
		"--keep-monthly", "24",
		"--yes",
	)

	// Verify output shows changes
	env.AssertOutputContains(stdout, "keep-daily: 7 -> 30")
	env.AssertOutputContains(stdout, "keep-weekly: 4 -> 12")
	env.AssertOutputContains(stdout, "keep-monthly: 6 -> 24")

	// Verify config was updated
	env.AssertFileContains(configPath, "KEEP_DAILY=30")
	env.AssertFileContains(configPath, "KEEP_WEEKLY=12")
	env.AssertFileContains(configPath, "KEEP_MONTHLY=24")
}

func TestEdit_UpdatesExcludes(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-excludes",
		"--paths", backupSource,
	)

	// Edit with new excludes
	stdout, _ := env.RunBakExpectSuccess(
		"edit",
		"--exclude", "*.bak",
		"--exclude", "*.swp",
		"--yes",
	)

	// Verify output shows changes
	env.AssertOutputContains(stdout, "excludes:")

	// Verify config was updated
	env.AssertFileContains(configPath, "*.bak")
	env.AssertFileContains(configPath, "*.swp")
}

func TestEdit_FailsWithoutSetup(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Edit without setup
	_, stderr, _ := env.RunBakExpectError(
		"edit",
		"--paths", "/new/path",
		"--yes",
	)

	env.AssertOutputContains(stderr, "not configured")
	env.AssertOutputContains(stderr, "Run 'bak setup' first")
}

func TestEdit_MultipleChanges(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-edit-multi",
		"--paths", backupSource,
		"--schedule", "daily",
	)

	// Edit multiple things at once
	newPath := "/tmp/backup-multi-edit"
	stdout, _ := env.RunBakExpectSuccess(
		"edit",
		"--paths", newPath,
		"--schedule", "hourly",
		"--keep-daily", "21",
		"--exclude", "*.tmp",
		"--yes",
	)

	// Verify all changes shown
	env.AssertOutputContains(stdout, "paths:")
	env.AssertOutputContains(stdout, "schedule: daily -> hourly")
	env.AssertOutputContains(stdout, "keep-daily: 7 -> 21")
	env.AssertOutputContains(stdout, "excludes:")

	// Verify config has all updates
	env.AssertFileContains(configPath, `BACKUP_PATHS="`+newPath+`"`)
	env.AssertFileContains(configPath, `BACKUP_SCHEDULE="hourly"`)
	env.AssertFileContains(configPath, "KEEP_DAILY=21")
	env.AssertFileContains(configPath, "*.tmp")
}
