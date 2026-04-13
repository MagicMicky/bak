// Package scheduler abstracts platform-specific task scheduling for backups.
package scheduler

// Scheduler is the interface for platform-specific backup scheduling.
type Scheduler interface {
	// Install sets up the scheduled backup task for the given schedule.
	Install(schedule string) error
	// Uninstall removes the scheduled backup task.
	Uninstall() error
	// UpdateSchedule updates the schedule of an existing task.
	UpdateSchedule(schedule string) error
	// Status returns a human-readable status of the scheduled task.
	Status() (string, error)
	// NextRun returns when the task will next trigger.
	NextRun() (string, error)
	// ViewLogs displays recent backup logs to stdout.
	ViewLogs(lines int) error
	// DryRunInfo returns file paths and contents that would be written during install.
	DryRunInfo(schedule, binaryPath string) []DryRunFile
	// DefaultBinaryPath returns the platform default installation path for bak.
	DefaultBinaryPath() string
}

// DryRunFile represents a file that would be written during scheduler installation.
type DryRunFile struct {
	Path    string
	Content string
}
