//go:build !windows

package scheduler

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	servicePath    = "/etc/systemd/system/backup.service"
	timerPath      = "/etc/systemd/system/backup.timer"
	defaultBinPath = "/usr/local/bin/bak"
)

const serviceTemplate = `[Unit]
Description=Restic backup
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s run-internal
Nice=10
IOSchedulingClass=idle
TimeoutStartSec=14400
`

const timerTemplate = `[Unit]
Description=Scheduled backup timer

[Timer]
OnCalendar=%s
RandomizedDelaySec=900
Persistent=true

[Install]
WantedBy=timers.target
`

type systemdScheduler struct{}

// New returns a Scheduler backed by systemd timers.
func New() Scheduler {
	return &systemdScheduler{}
}

func (s *systemdScheduler) DefaultBinaryPath() string {
	return defaultBinPath
}

func (s *systemdScheduler) Install(schedule string) error {
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = defaultBinPath
	}
	binaryPath, _ = filepath.Abs(binaryPath)

	if err := os.WriteFile(servicePath, []byte(generateService(binaryPath)), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	if err := os.WriteFile(timerPath, []byte(generateTimer(schedule)), 0644); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if err := exec.Command("systemctl", "enable", "backup.timer").Run(); err != nil {
		return fmt.Errorf("failed to enable timer: %w", err)
	}

	if err := exec.Command("systemctl", "start", "backup.timer").Run(); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}

	return nil
}

func (s *systemdScheduler) Uninstall() error {
	exec.Command("systemctl", "stop", "backup.timer").Run()
	exec.Command("systemctl", "disable", "backup.timer").Run()

	os.Remove(servicePath)
	os.Remove(timerPath)

	return exec.Command("systemctl", "daemon-reload").Run()
}

func (s *systemdScheduler) DryRunUninstallInfo() string {
	return fmt.Sprintf("Would run:\n"+
		"  systemctl stop backup.timer\n"+
		"  systemctl disable backup.timer\n"+
		"  rm %s\n"+
		"  rm %s\n"+
		"  systemctl daemon-reload", servicePath, timerPath)
}

func (s *systemdScheduler) UpdateSchedule(schedule string) error {
	if err := os.WriteFile(timerPath, []byte(generateTimer(schedule)), 0644); err != nil {
		return fmt.Errorf("failed to update timer: %w", err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	if err := exec.Command("systemctl", "restart", "backup.timer").Run(); err != nil {
		return fmt.Errorf("failed to restart timer: %w", err)
	}
	return nil
}

func (s *systemdScheduler) Status() (string, error) {
	out, err := exec.Command("systemctl", "status", "backup.timer", "--no-pager").CombinedOutput()
	return string(out), err
}

func (s *systemdScheduler) NextRun() (string, error) {
	out, err := exec.Command("systemctl", "show", "backup.timer",
		"--property=NextElapseUSecRealtime", "--value").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *systemdScheduler) ViewLogs(lines int) error {
	cmd := exec.Command("journalctl", "-u", "backup.service",
		"--no-pager", "-n", fmt.Sprintf("%d", lines))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *systemdScheduler) DryRunInfo(schedule, binaryPath string) []DryRunFile {
	if binaryPath == "" {
		binaryPath = defaultBinPath
	}
	return []DryRunFile{
		{Path: servicePath, Content: generateService(binaryPath)},
		{Path: timerPath, Content: generateTimer(schedule)},
	}
}

// generateOnCalendar converts a schedule string to a systemd OnCalendar specification.
func generateOnCalendar(schedule string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch schedule {
	case "daily":
		hour := 2 + r.Intn(6)
		minute := r.Intn(60)
		return fmt.Sprintf("*-*-* %02d:%02d:00", hour, minute)
	case "hourly":
		minute := r.Intn(60)
		return fmt.Sprintf("*-*-* *:%02d:00", minute)
	default:
		return schedule
	}
}

func generateService(binaryPath string) string {
	if binaryPath == "" {
		binaryPath = defaultBinPath
	}
	return fmt.Sprintf(serviceTemplate, binaryPath)
}

func generateTimer(schedule string) string {
	onCalendar := generateOnCalendar(schedule)
	return fmt.Sprintf(timerTemplate, onCalendar)
}
