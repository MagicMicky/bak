//go:build windows

package scheduler

import (
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
}

func TestWindowsDefaultBinaryPath(t *testing.T) {
	t.Parallel()

	s := &windowsScheduler{}
	want := `C:\Program Files\bak\bak.exe`
	if s.DefaultBinaryPath() != want {
		t.Errorf("DefaultBinaryPath() = %q, want %q", s.DefaultBinaryPath(), want)
	}
}
