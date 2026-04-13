//go:build windows

package scheduler

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	taskName       = "bak-backup"
	windowsBinPath = `C:\Program Files\bak\bak.exe`
)

type windowsScheduler struct{}

// New returns a Scheduler backed by Windows Task Scheduler.
func New() Scheduler {
	return &windowsScheduler{}
}

func (s *windowsScheduler) DefaultBinaryPath() string {
	return windowsBinPath
}

func (s *windowsScheduler) Install(schedule string) error {
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = windowsBinPath
	}
	binaryPath, _ = filepath.Abs(binaryPath)

	args, err := scheduleArgs(schedule)
	if err != nil {
		return err
	}

	cmdArgs := []string{"/Create", "/TN", taskName, "/TR", fmt.Sprintf(`"%s" run-internal`, binaryPath)}
	cmdArgs = append(cmdArgs, args...)
	// /F: overwrite existing task, /RL HIGHEST: run with elevated privileges,
	// /RU SYSTEM: run as SYSTEM so backups execute even when no user is logged in
	cmdArgs = append(cmdArgs, "/F", "/RL", "HIGHEST", "/RU", "SYSTEM")

	if err := exec.Command("schtasks", cmdArgs...).Run(); err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	return nil
}

func (s *windowsScheduler) Uninstall() error {
	// Ignore "task not found" errors, matching systemd behavior
	out, err := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").CombinedOutput()
	if err == nil {
		return nil
	}
	output := strings.ToLower(string(out))
	if strings.Contains(output, "cannot find the file") {
		return nil
	}
	return fmt.Errorf("failed to delete scheduled task: %w: %s", err, strings.TrimSpace(string(out)))
}

func (s *windowsScheduler) UpdateSchedule(schedule string) error {
	// Install uses /F flag which overwrites the existing task
	return s.Install(schedule)
}

func (s *windowsScheduler) Status() (string, error) {
	out, err := exec.Command("schtasks", "/Query", "/TN", taskName, "/V", "/FO", "LIST").CombinedOutput()
	return string(out), err
}

func (s *windowsScheduler) NextRun() (string, error) {
	// Use CSV format for locale-independent parsing (column order is fixed)
	out, err := exec.Command("schtasks", "/Query", "/TN", taskName, "/FO", "CSV", "/NH").CombinedOutput()
	if err != nil {
		return "", err
	}

	// CSV columns: "TaskName","Next Run Time","Status"
	line := strings.TrimSpace(string(out))
	fields := strings.Split(line, ",")
	if len(fields) >= 2 {
		return strings.Trim(fields[1], `"`), nil
	}

	return "", fmt.Errorf("could not determine next run time")
}

func (s *windowsScheduler) ViewLogs(lines int) error {
	return fmt.Errorf("logs command is not yet supported on Windows; check Windows Event Viewer for backup task output")
}

func (s *windowsScheduler) DryRunInfo(schedule, binaryPath string) []DryRunFile {
	if binaryPath == "" {
		binaryPath = windowsBinPath
	}

	args, err := scheduleArgs(schedule)
	if err != nil {
		args = []string{fmt.Sprintf("<error: %s>", err)}
	}

	cmdArgs := []string{"schtasks", "/Create", "/TN", taskName, "/TR", fmt.Sprintf(`"%s" run-internal`, binaryPath)}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "/F", "/RL", "HIGHEST", "/RU", "SYSTEM")

	return []DryRunFile{
		{
			Path:    "Windows Task Scheduler",
			Content: fmt.Sprintf("Command: %s\n", strings.Join(cmdArgs, " ")),
		},
	}
}

// scheduleArgs converts a schedule string to schtasks arguments.
func scheduleArgs(schedule string) ([]string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch schedule {
	case "daily":
		hour := 2 + r.Intn(6)
		minute := r.Intn(60)
		return []string{"/SC", "DAILY", "/ST", fmt.Sprintf("%02d:%02d", hour, minute)}, nil
	case "hourly":
		minute := r.Intn(60)
		return []string{"/SC", "HOURLY", "/MO", "1", "/ST", fmt.Sprintf("00:%02d", minute)}, nil
	default:
		return nil, fmt.Errorf("custom schedule %q is not supported on Windows; use 'daily' or 'hourly'", schedule)
	}
}
