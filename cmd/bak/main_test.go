package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tag     string
		wantErr bool
	}{
		{"valid simple", "webapp", false},
		{"valid with dash", "game-server", false},
		{"valid with underscore", "backup_prod", false},
		{"valid with numbers", "server01", false},
		{"valid mixed", "my-backup_2024", false},
		{"invalid space", "my server", true},
		{"invalid special char", "backup@home", true},
		{"invalid slash", "backup/test", true},
		{"invalid dot", "backup.conf", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTag(%q) error = %v, wantErr %v", tt.tag, err, tt.wantErr)
			}
		})
	}
}

func TestParsePaths(t *testing.T) {
	t.Parallel()

	// Create temp directories for testing
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing")
	if err := os.Mkdir(existingPath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	tests := []struct {
		name         string
		input        string
		wantPaths    int
		wantWarnings int
	}{
		{
			name:         "single existing path",
			input:        existingPath,
			wantPaths:    1,
			wantWarnings: 0,
		},
		{
			name:         "single non-existing path",
			input:        "/nonexistent/path/xyz",
			wantPaths:    1,
			wantWarnings: 1,
		},
		{
			name:         "multiple paths comma separated",
			input:        existingPath + ",/another/path",
			wantPaths:    2,
			wantWarnings: 1,
		},
		{
			name:         "paths with whitespace",
			input:        existingPath + " , /another/path ",
			wantPaths:    2,
			wantWarnings: 1,
		},
		{
			name:         "empty string",
			input:        "",
			wantPaths:    0,
			wantWarnings: 0,
		},
		{
			name:         "only commas",
			input:        ",,,",
			wantPaths:    0,
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			paths, warnings := parsePaths(tt.input)
			if len(paths) != tt.wantPaths {
				t.Errorf("parsePaths(%q) got %d paths, want %d", tt.input, len(paths), tt.wantPaths)
			}
			if len(warnings) != tt.wantWarnings {
				t.Errorf("parsePaths(%q) got %d warnings, want %d", tt.input, len(warnings), tt.wantWarnings)
			}
		})
	}
}

func TestPromptConfirm(t *testing.T) {
	// Note: promptConfirm reads from stdin, so we can't easily test it in unit tests
	// without mocking. This is left as a placeholder for potential integration testing.
	t.Skip("promptConfirm requires stdin mocking")
}
