//go:build windows

package scheduler

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magicmicky/bak/internal/config"
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

// logPath returns the path to the backup log file.
func logPath() string {
	return filepath.Join(config.DefaultConfigDir, "backup.log")
}

// taskCommand returns the /TR value that wraps bak with output redirection
// and a timestamp header before each run.
func taskCommand(binaryPath string) string {
	log := logPath()
	return fmt.Sprintf(`cmd /c "echo === %%DATE%% %%TIME%% === >> "%s" & "%s" run-internal >> "%s" 2>&1"`, log, binaryPath, log)
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

	cmdArgs := []string{"/Create", "/TN", taskName, "/TR", taskCommand(binaryPath)}
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

func (s *windowsScheduler) DryRunUninstallInfo() string {
	return fmt.Sprintf("Would run:\n  schtasks /Delete /TN %s /F", taskName)
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
	log := logPath()
	if _, err := os.Stat(log); os.IsNotExist(err) {
		return fmt.Errorf("no backup logs found at %s\nRun a backup first with 'bak now' or wait for the scheduled task", log)
	}

	// Read the last N lines from the log file
	file, err := os.Open(log)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	// Show last N lines
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	for _, line := range allLines[start:] {
		fmt.Println(line)
	}

	return nil
}

func (s *windowsScheduler) DryRunInfo(schedule, binaryPath string) []DryRunFile {
	if binaryPath == "" {
		binaryPath = windowsBinPath
	}

	args, err := scheduleArgs(schedule)
	if err != nil {
		args = []string{fmt.Sprintf("<error: %s>", err)}
	}

	cmdArgs := []string{"schtasks", "/Create", "/TN", taskName, "/TR", taskCommand(binaryPath)}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "/F", "/RL", "HIGHEST", "/RU", "SYSTEM")

	return []DryRunFile{
		{
			Path:    "Windows Task Scheduler",
			Content: fmt.Sprintf("Command: %s\n\nLogs: %s\n", strings.Join(cmdArgs, " "), logPath()),
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
