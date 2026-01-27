package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.KeepDaily != 7 {
		t.Errorf("KeepDaily = %d, want 7", cfg.KeepDaily)
	}
	if cfg.KeepWeekly != 4 {
		t.Errorf("KeepWeekly = %d, want 4", cfg.KeepWeekly)
	}
	if cfg.KeepMonthly != 6 {
		t.Errorf("KeepMonthly = %d, want 6", cfg.KeepMonthly)
	}
	if cfg.Schedule != "daily" {
		t.Errorf("Schedule = %q, want %q", cfg.Schedule, "daily")
	}
}

func TestRetentionTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "default values",
			cfg: Config{
				KeepHourly:  0,
				KeepDaily:   7,
				KeepWeekly:  4,
				KeepMonthly: 6,
				KeepYearly:  0,
			},
			want: "retain:h=0,d=7,w=4,m=6,y=0",
		},
		{
			name: "custom values",
			cfg: Config{
				KeepHourly:  24,
				KeepDaily:   30,
				KeepWeekly:  12,
				KeepMonthly: 24,
				KeepYearly:  5,
			},
			want: "retain:h=24,d=30,w=12,m=24,y=5",
		},
		{
			name: "all zeros",
			cfg: Config{
				KeepHourly:  0,
				KeepDaily:   0,
				KeepWeekly:  0,
				KeepMonthly: 0,
				KeepYearly:  0,
			},
			want: "retain:h=0,d=0,w=0,m=0,y=0",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.RetentionTag()
			if got != tt.want {
				t.Errorf("RetentionTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "backup.conf")

	original := &Config{
		Tag:         "test-server",
		Paths:       []string{"/home", "/etc"},
		Excludes:    []string{"*.tmp", "*.log"},
		KeepHourly:  12,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
		KeepYearly:  2,
		NotifyURL:   "https://notify.example.com/hook",
		Schedule:    "hourly",
	}

	if err := original.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Tag != original.Tag {
		t.Errorf("Tag = %q, want %q", loaded.Tag, original.Tag)
	}
	if len(loaded.Paths) != len(original.Paths) {
		t.Errorf("Paths length = %d, want %d", len(loaded.Paths), len(original.Paths))
	}
	for i, p := range loaded.Paths {
		if p != original.Paths[i] {
			t.Errorf("Paths[%d] = %q, want %q", i, p, original.Paths[i])
		}
	}
	if loaded.KeepHourly != original.KeepHourly {
		t.Errorf("KeepHourly = %d, want %d", loaded.KeepHourly, original.KeepHourly)
	}
	if loaded.Schedule != original.Schedule {
		t.Errorf("Schedule = %q, want %q", loaded.Schedule, original.Schedule)
	}
}

func TestLoadNonExistent(t *testing.T) {
	t.Parallel()

	_, err := Load("/nonexistent/path/config.conf")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}

func TestExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.conf")

	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !Exists(existingFile) {
		t.Error("Exists() = false for existing file, want true")
	}

	if Exists(filepath.Join(tmpDir, "nonexistent.conf")) {
		t.Error("Exists() = true for nonexistent file, want false")
	}
}

func TestLoadWithComments(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "backup.conf")

	content := `# This is a comment
BACKUP_TAG="myserver"
# Another comment
BACKUP_PATHS="/home,/var"
KEEP_DAILY=14

# Empty lines are fine

BACKUP_SCHEDULE="daily"
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Tag != "myserver" {
		t.Errorf("Tag = %q, want %q", cfg.Tag, "myserver")
	}
	if cfg.KeepDaily != 14 {
		t.Errorf("KeepDaily = %d, want 14", cfg.KeepDaily)
	}
}
