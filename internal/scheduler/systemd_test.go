//go:build !windows

package scheduler

import (
	"strings"
	"testing"
)

func TestGenerateOnCalendar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schedule string
		wantFmt  string
	}{
		{
			name:     "daily generates hour:minute format",
			schedule: "daily",
			wantFmt:  "*-*-* ",
		},
		{
			name:     "hourly generates minute format",
			schedule: "hourly",
			wantFmt:  "*-*-* *:",
		},
		{
			name:     "custom cron expression passthrough",
			schedule: "*-*-* 02:00:00",
			wantFmt:  "*-*-* 02:00:00",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := generateOnCalendar(tt.schedule)
			if !strings.Contains(result, tt.wantFmt) {
				t.Errorf("generateOnCalendar(%q) = %q, want to contain %q", tt.schedule, result, tt.wantFmt)
			}
		})
	}
}

func TestGenerateService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		binaryPath string
		wantExec   string
	}{
		{
			name:       "custom binary path",
			binaryPath: "/opt/bin/bak",
			wantExec:   "ExecStart=/opt/bin/bak run-internal",
		},
		{
			name:       "empty uses default",
			binaryPath: "",
			wantExec:   "ExecStart=/usr/local/bin/bak run-internal",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := generateService(tt.binaryPath)
			if !strings.Contains(result, tt.wantExec) {
				t.Errorf("generateService(%q) missing %q", tt.binaryPath, tt.wantExec)
			}
			if !strings.Contains(result, "[Unit]") {
				t.Error("generateService missing [Unit] section")
			}
			if !strings.Contains(result, "[Service]") {
				t.Error("generateService missing [Service] section")
			}
		})
	}
}

func TestGenerateTimer(t *testing.T) {
	t.Parallel()

	result := generateTimer("daily")

	checks := []string{"[Unit]", "[Timer]", "[Install]", "OnCalendar=", "RandomizedDelaySec=", "Persistent=true"}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("generateTimer missing %q", want)
		}
	}
}

func TestGenerateTimerWithCustomSchedule(t *testing.T) {
	t.Parallel()

	result := generateTimer("*-*-* 12:00:00")
	if !strings.Contains(result, "OnCalendar=*-*-* 12:00:00") {
		t.Errorf("generateTimer with custom schedule missing exact schedule, got: %s", result)
	}
}

func TestDryRunInfo(t *testing.T) {
	t.Parallel()

	s := &systemdScheduler{}
	files := s.DryRunInfo("daily", "/opt/bin/bak")

	if len(files) != 2 {
		t.Fatalf("DryRunInfo returned %d files, want 2", len(files))
	}

	if files[0].Path != servicePath {
		t.Errorf("DryRunInfo[0].Path = %q, want %q", files[0].Path, servicePath)
	}
	if !strings.Contains(files[0].Content, "ExecStart=/opt/bin/bak run-internal") {
		t.Error("DryRunInfo service content missing ExecStart")
	}

	if files[1].Path != timerPath {
		t.Errorf("DryRunInfo[1].Path = %q, want %q", files[1].Path, timerPath)
	}
	if !strings.Contains(files[1].Content, "OnCalendar=") {
		t.Error("DryRunInfo timer content missing OnCalendar")
	}
}

func TestDefaultBinaryPath(t *testing.T) {
	t.Parallel()

	s := &systemdScheduler{}
	if s.DefaultBinaryPath() != "/usr/local/bin/bak" {
		t.Errorf("DefaultBinaryPath() = %q, want %q", s.DefaultBinaryPath(), "/usr/local/bin/bak")
	}
}
