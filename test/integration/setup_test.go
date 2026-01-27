//go:build integration

package integration

import (
	"testing"
)

func TestSetup_BasicConfiguration(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run setup
	stdout, _ := env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-basic",
		"--paths", backupSource,
	)

	// Verify output
	env.AssertOutputContains(stdout, "Setup complete!")
	env.AssertOutputContains(stdout, "Tag:      test-basic")

	// Verify config file exists and has correct content
	if !env.FileExists(configPath) {
		t.Fatal("config file was not created")
	}
	env.AssertFileContains(configPath, `BACKUP_TAG="test-basic"`)
	env.AssertFileContains(configPath, `BACKUP_PATHS="/tmp/backup-source"`)

	// Verify systemd service file exists
	if !env.FileExists(servicePath) {
		t.Fatal("service file was not created")
	}
	env.AssertFileContains(servicePath, "ExecStart=/usr/local/bin/bak run-internal")

	// Verify systemd timer file exists
	if !env.FileExists(timerPath) {
		t.Fatal("timer file was not created")
	}
	env.AssertFileContains(timerPath, "OnCalendar=")

	// Verify timer is enabled and active
	if !env.TimerIsEnabled() {
		t.Error("timer is not enabled")
	}
	if !env.TimerIsActive() {
		t.Error("timer is not active")
	}
}

func TestSetup_CustomSchedule(t *testing.T) {
	tests := []struct {
		name            string
		schedule        string
		expectedPattern string
	}{
		{
			name:            "daily schedule",
			schedule:        "daily",
			expectedPattern: "*-*-*", // daily generates *-*-* HH:MM:00
		},
		{
			name:            "hourly schedule",
			schedule:        "hourly",
			expectedPattern: "*-*-* *:", // hourly generates *-*-* *:MM:00
		},
		{
			name:            "cron expression",
			schedule:        "*-*-* 03:30:00",
			expectedPattern: "*-*-* 03:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewTestEnv(t)
			defer env.Cleanup()

			// Run setup with custom schedule
			env.RunBakExpectSuccess(
				"setup",
				"--tag", "test-schedule",
				"--paths", backupSource,
				"--schedule", tt.schedule,
			)

			// Verify schedule in config
			env.AssertFileContains(configPath, `BACKUP_SCHEDULE="`+tt.schedule+`"`)

			// Verify timer has OnCalendar entry
			env.AssertFileContains(timerPath, "OnCalendar=")
			env.AssertFileContains(timerPath, tt.expectedPattern)
		})
	}
}

func TestSetup_ExistingConfigBlocked(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run first setup
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-first",
		"--paths", backupSource,
	)

	// Second setup without --force should fail
	_, stderr, _ := env.RunBakExpectError(
		"setup",
		"--tag", "test-second",
		"--paths", backupSource,
	)
	env.AssertOutputContains(stderr, "configuration already exists")

	// Verify original config is unchanged
	env.AssertFileContains(configPath, `BACKUP_TAG="test-first"`)

	// Setup with --force should succeed
	stdout, _ := env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-force",
		"--paths", backupSource,
		"--force",
	)
	env.AssertOutputContains(stdout, "Setup complete!")

	// Verify config was updated
	env.AssertFileContains(configPath, `BACKUP_TAG="test-force"`)
}

func TestSetup_RetentionFlags(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run setup with custom retention
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-retention",
		"--paths", backupSource,
		"--keep-hourly", "24",
		"--keep-daily", "14",
		"--keep-weekly", "8",
		"--keep-monthly", "12",
		"--keep-yearly", "3",
	)

	// Verify retention values in config
	env.AssertFileContains(configPath, "KEEP_HOURLY=24")
	env.AssertFileContains(configPath, "KEEP_DAILY=14")
	env.AssertFileContains(configPath, "KEEP_WEEKLY=8")
	env.AssertFileContains(configPath, "KEEP_MONTHLY=12")
	env.AssertFileContains(configPath, "KEEP_YEARLY=3")
}

func TestSetup_ExcludePatterns(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run setup with exclude patterns
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-excludes",
		"--paths", backupSource,
		"--exclude", "*.log",
		"--exclude", "*.tmp",
	)

	// Verify excludes in config
	env.AssertFileContains(configPath, `BACKUP_EXCLUDES="*.log *.tmp"`)
}

func TestSetup_InvalidTag(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run setup with invalid tag
	_, stderr, _ := env.RunBakExpectError(
		"setup",
		"--tag", "invalid tag!", // contains space and special char
		"--paths", backupSource,
	)
	env.AssertOutputContains(stderr, "tag must contain only alphanumeric")
}
