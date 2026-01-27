//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	bakBinary    = "/usr/local/bin/bak"
	configPath   = "/etc/backup/backup.conf"
	envPath      = "/etc/backup/env"
	servicePath  = "/etc/systemd/system/backup.service"
	timerPath    = "/etc/systemd/system/backup.timer"
	resticRepo   = "/tmp/restic-repo"
	backupSource = "/tmp/backup-source"
)

// TestEnv provides a clean test environment for integration tests
type TestEnv struct {
	t *testing.T
}

// NewTestEnv creates a new test environment with clean state
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	env := &TestEnv{t: t}
	env.cleanup()
	env.setupEnvFile()
	env.setupTestData()
	return env
}

// cleanup removes config, stops timer, and removes systemd units
func (e *TestEnv) cleanup() {
	e.t.Helper()

	// Stop and disable timer (ignore errors - may not exist)
	exec.Command("systemctl", "stop", "backup.timer").Run()
	exec.Command("systemctl", "disable", "backup.timer").Run()

	// Remove config files
	os.Remove(configPath)
	os.Remove(envPath)

	// Remove systemd unit files
	os.Remove(servicePath)
	os.Remove(timerPath)

	// Reload systemd to clear cached units
	exec.Command("systemctl", "daemon-reload").Run()
}

// setupEnvFile creates the environment file with restic credentials
func (e *TestEnv) setupEnvFile() {
	e.t.Helper()

	content := `RESTIC_REPOSITORY=/tmp/restic-repo
RESTIC_PASSWORD=test-password
`
	if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
		e.t.Fatalf("failed to create env directory: %v", err)
	}
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("failed to write env file: %v", err)
	}
}

// setupTestData creates sample files for backup testing
func (e *TestEnv) setupTestData() {
	e.t.Helper()

	// Create backup source directory if it doesn't exist
	if err := os.MkdirAll(backupSource, 0755); err != nil {
		e.t.Fatalf("failed to create backup source: %v", err)
	}

	// Create test files
	files := map[string]string{
		"file1.txt":         "test file 1 content",
		"file2.txt":         "test file 2 content",
		"subdir/nested.txt": "nested file content",
	}

	for name, content := range files {
		path := filepath.Join(backupSource, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			e.t.Fatalf("failed to create directory for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			e.t.Fatalf("failed to write %s: %v", name, err)
		}
	}
}

// Cleanup removes all test artifacts - call in defer
func (e *TestEnv) Cleanup() {
	e.t.Helper()
	e.cleanup()
}

// RunBak executes the bak command with the given arguments
// Returns stdout, stderr, and any error
func (e *TestEnv) RunBak(args ...string) (string, string, error) {
	e.t.Helper()

	cmd := exec.Command(bakBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// RunBakExpectSuccess runs bak and fails the test if it errors
func (e *TestEnv) RunBakExpectSuccess(args ...string) (string, string) {
	e.t.Helper()
	stdout, stderr, err := e.RunBak(args...)
	if err != nil {
		e.t.Fatalf("bak %v failed: %v\nstdout: %s\nstderr: %s", args, err, stdout, stderr)
	}
	return stdout, stderr
}

// RunBakExpectError runs bak and fails the test if it succeeds
func (e *TestEnv) RunBakExpectError(args ...string) (string, string, error) {
	e.t.Helper()
	stdout, stderr, err := e.RunBak(args...)
	if err == nil {
		e.t.Fatalf("bak %v expected error but succeeded\nstdout: %s\nstderr: %s", args, stdout, stderr)
	}
	return stdout, stderr, err
}

// FileExists checks if a file exists
func (e *TestEnv) FileExists(path string) bool {
	e.t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads and returns file contents
func (e *TestEnv) ReadFile(path string) string {
	e.t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		e.t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(content)
}

// AssertFileContains checks that a file contains the expected string
func (e *TestEnv) AssertFileContains(path, expected string) {
	e.t.Helper()
	content := e.ReadFile(path)
	if !strings.Contains(content, expected) {
		e.t.Errorf("file %s does not contain %q\nactual content:\n%s", path, expected, content)
	}
}

// AssertOutputContains checks that output contains the expected string
func (e *TestEnv) AssertOutputContains(output, expected string) {
	e.t.Helper()
	if !strings.Contains(output, expected) {
		e.t.Errorf("output does not contain %q\nactual output:\n%s", expected, output)
	}
}

// TimerIsActive checks if the backup timer is active
func (e *TestEnv) TimerIsActive() bool {
	e.t.Helper()
	err := exec.Command("systemctl", "is-active", "--quiet", "backup.timer").Run()
	return err == nil
}

// TimerIsEnabled checks if the backup timer is enabled
func (e *TestEnv) TimerIsEnabled() bool {
	e.t.Helper()
	err := exec.Command("systemctl", "is-enabled", "--quiet", "backup.timer").Run()
	return err == nil
}

// GetSnapshotCount returns the number of restic snapshots with the given tag
func (e *TestEnv) GetSnapshotCount(tag string) int {
	e.t.Helper()

	cmd := exec.Command("restic", "snapshots", "--tag", tag, "--json")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Simple count of snapshot entries - look for "short_id" occurrences
	return strings.Count(string(output), `"short_id"`)
}
