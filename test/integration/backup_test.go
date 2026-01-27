//go:build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"
)

func TestNow_PerformsBackup(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-now",
		"--paths", backupSource,
	)

	// Get initial snapshot count
	initialCount := env.GetSnapshotCount("test-now")

	// Run backup
	stdout, _ := env.RunBakExpectSuccess("now")
	env.AssertOutputContains(stdout, "Backup completed successfully!")

	// Verify new snapshot was created
	newCount := env.GetSnapshotCount("test-now")
	if newCount <= initialCount {
		t.Errorf("expected snapshot count to increase, got %d (was %d)", newCount, initialCount)
	}

	// Verify snapshot has correct tag using restic directly
	cmd := exec.Command("restic", "snapshots", "--tag", "test-now", "--json")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}
	if !strings.Contains(string(output), `"test-now"`) {
		t.Error("snapshot does not have expected tag")
	}
}

func TestNow_FailsWithoutSetup(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Run backup without setup
	_, stderr, _ := env.RunBakExpectError("now")
	env.AssertOutputContains(stderr, "not configured")
	env.AssertOutputContains(stderr, "Run 'bak setup' first")
}

func TestRunInternal_Works(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup first
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-internal",
		"--paths", backupSource,
	)

	// Get initial snapshot count
	initialCount := env.GetSnapshotCount("test-internal")

	// Run internal command (what systemd calls)
	stdout, stderr, err := env.RunBak("run-internal")
	if err != nil {
		t.Fatalf("run-internal failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	// Verify new snapshot was created
	newCount := env.GetSnapshotCount("test-internal")
	if newCount <= initialCount {
		t.Errorf("expected snapshot count to increase, got %d (was %d)", newCount, initialCount)
	}
}

func TestBackup_WithExcludes(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a file that should be excluded
	excludedPath := backupSource + "/excluded.log"
	if err := exec.Command("sh", "-c", "echo 'log content' > "+excludedPath).Run(); err != nil {
		t.Fatalf("failed to create excluded file: %v", err)
	}

	// Setup with exclude pattern
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-excludes-backup",
		"--paths", backupSource,
		"--exclude", "*.log",
	)

	// Run backup
	env.RunBakExpectSuccess("now")

	// List files in latest snapshot
	cmd := exec.Command("restic", "ls", "latest", "--tag", "test-excludes-backup")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to list snapshot contents: %v", err)
	}

	// Verify excluded file is not in snapshot
	if strings.Contains(string(output), "excluded.log") {
		t.Error("excluded file should not be in snapshot")
	}

	// Verify regular files are in snapshot
	if !strings.Contains(string(output), "file1.txt") {
		t.Error("file1.txt should be in snapshot")
	}
}

func TestBackup_MultiplePaths(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create additional test directory
	extraPath := "/tmp/backup-source-extra"
	exec.Command("mkdir", "-p", extraPath).Run()
	exec.Command("sh", "-c", "echo 'extra content' > "+extraPath+"/extra.txt").Run()

	// Setup with multiple paths
	env.RunBakExpectSuccess(
		"setup",
		"--tag", "test-multi-path",
		"--paths", backupSource+","+extraPath,
	)

	// Run backup
	env.RunBakExpectSuccess("now")

	// List files in latest snapshot
	cmd := exec.Command("restic", "ls", "latest", "--tag", "test-multi-path")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to list snapshot contents: %v", err)
	}

	// Verify files from both paths are in snapshot
	if !strings.Contains(string(output), "file1.txt") {
		t.Error("file1.txt from first path should be in snapshot")
	}
	if !strings.Contains(string(output), "extra.txt") {
		t.Error("extra.txt from second path should be in snapshot")
	}
}
