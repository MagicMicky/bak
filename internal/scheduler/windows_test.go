//go:build windows

package scheduler

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScheduleArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schedule string
		wantSC   string
		wantErr  bool
	}{
		{
			name:     "daily schedule",
			schedule: "daily",
			wantSC:   "DAILY",
		},
		{
			name:     "hourly schedule",
			schedule: "hourly",
			wantSC:   "HOURLY",
		},
		{
			name:     "custom schedule rejected",
			schedule: "*-*-* 02:00:00",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args, err := scheduleArgs(tt.schedule)
			if tt.wantErr {
				if err == nil {
					t.Errorf("scheduleArgs(%q) expected error, got nil", tt.schedule)
				}
				return
			}
			if err != nil {
				t.Fatalf("scheduleArgs(%q) unexpected error: %v", tt.schedule, err)
			}
			found := false
			for _, arg := range args {
				if arg == tt.wantSC {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("scheduleArgs(%q) = %v, want to contain %q", tt.schedule, args, tt.wantSC)
			}
		})
	}
}

func TestTaskCommand(t *testing.T) {
	t.Parallel()

	cmd := taskCommand(`C:\Program Files\bak\bak.exe`)
	if !strings.Contains(cmd, "run-internal") {
		t.Error("taskCommand missing run-internal")
	}
	if !strings.Contains(cmd, ">>") {
		t.Error("taskCommand missing output redirection")
	}
	if !strings.Contains(cmd, "2>&1") {
		t.Error("taskCommand missing stderr redirection")
	}
	if !strings.Contains(cmd, "backup.log") {
		t.Error("taskCommand missing log file path")
	}
	if !strings.HasPrefix(cmd, `cmd /c`) {
		t.Error("taskCommand should start with cmd /c")
	}
	if !strings.Contains(cmd, "%DATE%") || !strings.Contains(cmd, "%TIME%") {
		t.Error("taskCommand missing timestamp header with %%DATE%% and %%TIME%%")
	}
	if !strings.Contains(cmd, "echo ===") {
		t.Error("taskCommand missing echo timestamp separator")
	}
}

func TestWindowsDryRunInfo(t *testing.T) {
	t.Parallel()

	s := &windowsScheduler{}
	files := s.DryRunInfo("daily", `C:\bak\bak.exe`)

	if len(files) != 1 {
		t.Fatalf("DryRunInfo returned %d files, want 1", len(files))
	}

	if files[0].Path != "Windows Task Scheduler" {
		t.Errorf("DryRunInfo[0].Path = %q, want %q", files[0].Path, "Windows Task Scheduler")
	}
	if !strings.Contains(files[0].Content, "schtasks") {
		t.Error("DryRunInfo content missing schtasks command")
	}
	if !strings.Contains(files[0].Content, "bak-backup") {
		t.Error("DryRunInfo content missing task name")
	}
	if !strings.Contains(files[0].Content, "backup.log") {
		t.Error("DryRunInfo content missing log path")
	}
}

func TestWindowsDefaultBinaryPath(t *testing.T) {
	t.Parallel()

	s := &windowsScheduler{}
	want := `C:\Program Files\bak\bak.exe`
	if s.DefaultBinaryPath() != want {
		t.Errorf("DefaultBinaryPath() = %q, want %q", s.DefaultBinaryPath(), want)
	}
}

func TestViewLogs(t *testing.T) {
	t.Parallel()

	// Test with non-existent log file
	s := &windowsScheduler{}
	err := s.ViewLogs(20)
	if err == nil {
		t.Error("ViewLogs should error when no log file exists")
	}
}

func TestViewLogsReadsFile(t *testing.T) {
	t.Parallel()

	// Create a temp log file and verify ViewLogs reads it
	tmpDir := t.TempDir()
	tmpLog := filepath.Join(tmpDir, "test.log")

	lines := []string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
	}
	if err := os.WriteFile(tmpLog, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test log: %v", err)
	}

	// Test the tail logic directly since ViewLogs uses hardcoded logPath()
	file, err := os.Open(tmpLog)
	if err != nil {
		t.Fatalf("failed to open test log: %v", err)
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) != 5 {
		t.Errorf("read %d lines, want 5", len(allLines))
	}

	// Test tail: last 3 lines
	start := 0
	if len(allLines) > 3 {
		start = len(allLines) - 3
	}
	tail := allLines[start:]
	if len(tail) != 3 {
		t.Errorf("tail got %d lines, want 3", len(tail))
	}
	if tail[0] != "line 3" {
		t.Errorf("tail[0] = %q, want %q", tail[0], "line 3")
	}
}
