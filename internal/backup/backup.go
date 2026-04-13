// Package backup handles the execution of restic backup commands.
package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/magicmicky/bak/internal/config"
)

// ErrResticNotFound is returned when restic binary is not found in PATH.
var ErrResticNotFound = errors.New("restic is not installed or not found in PATH")

// ResticAvailable checks if the restic binary is available in PATH.
func ResticAvailable() bool {
	_, err := exec.LookPath("restic")
	return err == nil
}

// RequireRestic returns an error if restic is not available.
func RequireRestic() error {
	if !ResticAvailable() {
		return ErrResticNotFound
	}
	return nil
}

// Snapshot represents a restic snapshot from JSON output.
type Snapshot struct {
	ShortID  string   `json:"short_id"`
	ID       string   `json:"id"`
	Time     string   `json:"time"`
	Hostname string   `json:"hostname"`
	Paths    []string `json:"paths"`
	Tags     []string `json:"tags"`
}

// Runner handles backup execution
type Runner struct {
	Config  *config.Config
	Verbose bool // Pass -v to restic for detailed file output
	DryRun  bool
}

// NewRunner creates a new backup runner
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{
		Config: cfg,
	}
}

// Run executes the backup with the configured settings
func (r *Runner) Run() error {
	// Build restic command arguments
	args := []string{"backup"}

	// Add dry-run flag if requested
	if r.DryRun {
		args = append(args, "-n")
	}

	// Add verbosity flags
	// -vv shows all files being processed (useful for dry-run validation)
	// Without verbose, restic shows progress bar for normal backups
	// and just summary for dry-run
	if r.Verbose {
		args = append(args, "-vv")
	}

	// Add paths
	args = append(args, r.Config.Paths...)

	// Add tags
	args = append(args, "--tag", r.Config.Tag)
	args = append(args, "--tag", r.Config.RetentionTag())

	// Add standard options
	args = append(args, "--exclude-caches")
	args = append(args, "--exclude-if-present", ".nobackup")
	if runtime.GOOS != "windows" {
		// --one-file-system is not supported on Windows (no device IDs)
		args = append(args, "--one-file-system")
	}

	// Add exclude patterns
	for _, exclude := range r.Config.Excludes {
		args = append(args, "--exclude", exclude)
	}

	// Create command
	cmd := exec.Command("restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ListSnapshots returns recent snapshots for the configured tag
func (r *Runner) ListSnapshots(limit int) error {
	args := []string{"snapshots", "--tag", r.Config.Tag}
	if limit > 0 {
		args = append(args, "--latest", fmt.Sprintf("%d", limit))
	}

	cmd := exec.Command("restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ListSnapshotsJSON returns snapshots as structured data.
func (r *Runner) ListSnapshotsJSON(limit int) ([]Snapshot, error) {
	args := []string{"snapshots", "--tag", r.Config.Tag, "--json"}
	if limit > 0 {
		args = append(args, "--latest", fmt.Sprintf("%d", limit))
	}

	cmd := exec.Command("restic", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(output, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot data: %w", err)
	}

	// Filter to only include snapshots with matching tag
	// This is a defensive check since restic's --tag filter may not work
	// correctly with --latest in some versions
	var filtered []Snapshot
	for _, s := range snapshots {
		for _, tag := range s.Tags {
			if tag == r.Config.Tag {
				filtered = append(filtered, s)
				break
			}
		}
	}

	return filtered, nil
}

// CheckRepository verifies the repository is accessible
func CheckRepository() error {
	cmd := exec.Command("restic", "cat", "config")
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
