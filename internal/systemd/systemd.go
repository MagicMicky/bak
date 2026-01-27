// Package systemd handles generation and management of systemd units for scheduled backups.
package systemd

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// ServicePath is the path to the systemd service file
	ServicePath = "/etc/systemd/system/backup.service"
	// TimerPath is the path to the systemd timer file
	TimerPath = "/etc/systemd/system/backup.timer"
	// BinaryPath is the expected installation path of the bak binary
	BinaryPath = "/usr/local/bin/bak"
)

// ServiceTemplate is the systemd service unit template
const ServiceTemplate = `[Unit]
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

// TimerTemplate is the systemd timer unit template
const TimerTemplate = `[Unit]
Description=Scheduled backup timer

[Timer]
OnCalendar=%s
RandomizedDelaySec=900
Persistent=true

[Install]
WantedBy=timers.target
`

// GenerateOnCalendar converts a schedule string to a systemd OnCalendar specification
func GenerateOnCalendar(schedule string) string {
	// Seed random for consistent but random times
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch schedule {
	case "daily":
		// Random hour between 2-7 AM, random minute
		hour := 2 + r.Intn(6)
		minute := r.Intn(60)
		return fmt.Sprintf("*-*-* %02d:%02d:00", hour, minute)
	case "hourly":
		// Random minute each hour
		minute := r.Intn(60)
		return fmt.Sprintf("*-*-* *:%02d:00", minute)
	default:
		// Assume it's a valid OnCalendar spec
		return schedule
	}
}

// GenerateService returns the service unit content for the given binary path.
func GenerateService(binaryPath string) string {
	if binaryPath == "" {
		binaryPath = BinaryPath
	}
	return fmt.Sprintf(ServiceTemplate, binaryPath)
}

// GenerateTimer returns the timer unit content for the given schedule.
func GenerateTimer(schedule string) string {
	onCalendar := GenerateOnCalendar(schedule)
	return fmt.Sprintf(TimerTemplate, onCalendar)
}

// WriteService creates the systemd service file
func WriteService(binaryPath string) error {
	return os.WriteFile(ServicePath, []byte(GenerateService(binaryPath)), 0644)
}

// WriteTimer creates the systemd timer file
func WriteTimer(schedule string) error {
	return os.WriteFile(TimerPath, []byte(GenerateTimer(schedule)), 0644)
}

// ReloadDaemon runs systemctl daemon-reload
func ReloadDaemon() error {
	return exec.Command("systemctl", "daemon-reload").Run()
}

// EnableTimer enables the backup timer
func EnableTimer() error {
	return exec.Command("systemctl", "enable", "backup.timer").Run()
}

// StartTimer starts the backup timer
func StartTimer() error {
	return exec.Command("systemctl", "start", "backup.timer").Run()
}

// RestartTimer restarts the backup timer
func RestartTimer() error {
	return exec.Command("systemctl", "restart", "backup.timer").Run()
}

// StopTimer stops the backup timer
func StopTimer() error {
	return exec.Command("systemctl", "stop", "backup.timer").Run()
}

// DisableTimer disables the backup timer
func DisableTimer() error {
	return exec.Command("systemctl", "disable", "backup.timer").Run()
}

// TimerStatus returns the status of the backup timer
func TimerStatus() (string, error) {
	out, err := exec.Command("systemctl", "status", "backup.timer", "--no-pager").CombinedOutput()
	return string(out), err
}

// NextRun returns when the timer will next trigger
func NextRun() (string, error) {
	out, err := exec.Command("systemctl", "show", "backup.timer", "--property=NextElapseUSecRealtime", "--value").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Install sets up the systemd service and timer
func Install(schedule string) error {
	// Find the actual binary path
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = BinaryPath
	}
	binaryPath, _ = filepath.Abs(binaryPath)

	// Write service file
	if err := WriteService(binaryPath); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Write timer file
	if err := WriteTimer(schedule); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	// Reload systemd
	if err := ReloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable timer
	if err := EnableTimer(); err != nil {
		return fmt.Errorf("failed to enable timer: %w", err)
	}

	// Start timer
	if err := StartTimer(); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}

	return nil
}

// Uninstall removes the systemd service and timer
func Uninstall() error {
	// Stop and disable timer
	StopTimer()
	DisableTimer()

	// Remove files
	os.Remove(ServicePath)
	os.Remove(TimerPath)

	// Reload systemd
	return ReloadDaemon()
}
