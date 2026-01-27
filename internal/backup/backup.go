// Package backup handles the execution of restic backup commands.
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/magicmicky/bak/internal/config"
)

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
	Verbose bool
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

	// Add paths
	args = append(args, r.Config.Paths...)

	// Add tags
	args = append(args, "--tag", r.Config.Tag)
	args = append(args, "--tag", r.Config.RetentionTag())

	// Add standard options
	args = append(args, "--exclude-caches")
	args = append(args, "--exclude-if-present", ".nobackup")
	args = append(args, "--one-file-system")

	// Add exclude patterns
	for _, exclude := range r.Config.Excludes {
		args = append(args, "--exclude", exclude)
	}

	// Create command
	cmd := exec.Command("restic", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.Verbose {
		fmt.Printf("Executing: restic %v\n", args)
	}

	return cmd.Run()
}

// ListSnapshots returns recent snapshots for the configured tag
func (r *Runner) ListSnapshots(limit int) error {
	args := []string{"snapshots", "--tag", r.Config.Tag}
	if limit > 0 {
		args = append(args, "--last", fmt.Sprintf("%d", limit))
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
		args = append(args, "--last", fmt.Sprintf("%d", limit))
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

	return snapshots, nil
}

// CheckRepository verifies the repository is accessible
func CheckRepository() error {
	cmd := exec.Command("restic", "cat", "config")
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
