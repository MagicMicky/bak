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
	cmdArgs = append(cmdArgs, "/F", "/RL", "HIGHEST")

	if err := exec.Command("schtasks", cmdArgs...).Run(); err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	return nil
}

func (s *windowsScheduler) Uninstall() error {
	// Ignore errors (task may not exist), matching systemd behavior
	exec.Command("schtasks", "/Delete", "/TN", taskName, "/F").Run()
	return nil
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
	out, err := exec.Command("schtasks", "/Query", "/TN", taskName, "/FO", "LIST").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Next Run Time:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Next Run Time:")), nil
		}
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
	cmdArgs = append(cmdArgs, "/F", "/RL", "HIGHEST")

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
		return []string{"/SC", "HOURLY"}, nil
	default:
		return nil, fmt.Errorf("custom schedule %q is not supported on Windows; use 'daily' or 'hourly'", schedule)
	}
}
