package systemd

import (
	"strings"
	"testing"
)

func TestGenerateOnCalendar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schedule string
		wantFmt  string // format to check for
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

			result := GenerateOnCalendar(tt.schedule)

			if !strings.Contains(result, tt.wantFmt) {
				t.Errorf("GenerateOnCalendar(%q) = %q, want to contain %q", tt.schedule, result, tt.wantFmt)
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

			result := GenerateService(tt.binaryPath)

			if !strings.Contains(result, tt.wantExec) {
				t.Errorf("GenerateService(%q) missing %q", tt.binaryPath, tt.wantExec)
			}
			if !strings.Contains(result, "[Unit]") {
				t.Error("GenerateService missing [Unit] section")
			}
			if !strings.Contains(result, "[Service]") {
				t.Error("GenerateService missing [Service] section")
			}
		})
	}
}

func TestGenerateTimer(t *testing.T) {
	t.Parallel()

	result := GenerateTimer("daily")

	if !strings.Contains(result, "[Unit]") {
		t.Error("GenerateTimer missing [Unit] section")
	}
	if !strings.Contains(result, "[Timer]") {
		t.Error("GenerateTimer missing [Timer] section")
	}
	if !strings.Contains(result, "[Install]") {
		t.Error("GenerateTimer missing [Install] section")
	}
	if !strings.Contains(result, "OnCalendar=") {
		t.Error("GenerateTimer missing OnCalendar")
	}
	if !strings.Contains(result, "RandomizedDelaySec=") {
		t.Error("GenerateTimer missing RandomizedDelaySec")
	}
	if !strings.Contains(result, "Persistent=true") {
		t.Error("GenerateTimer missing Persistent=true")
	}
}

func TestGenerateTimerWithCustomSchedule(t *testing.T) {
	t.Parallel()

	result := GenerateTimer("*-*-* 12:00:00")

	if !strings.Contains(result, "OnCalendar=*-*-* 12:00:00") {
		t.Errorf("GenerateTimer with custom schedule missing exact schedule, got: %s", result)
	}
}
