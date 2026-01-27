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

func TestValidateCredentials(t *testing.T) {
	// Test that validateCredentials properly sets environment variables and calls restic
	// Note: This test requires restic to be installed and will fail with invalid credentials
	t.Parallel()

	// Test with obviously invalid credentials
	err := validateCredentials("rest:https://invalid.example.com:8000", "wrongpassword")
	if err == nil {
		t.Error("validateCredentials() expected error for invalid credentials, got nil")
	}
}

func TestPromptInput(t *testing.T) {
	// Note: promptInput reads from stdin, so we can't easily test it in unit tests
	// without mocking. This is left as a placeholder for potential integration testing.
	t.Skip("promptInput requires stdin mocking")
}

func TestPromptPassword(t *testing.T) {
	// Note: promptPassword reads from stdin with terminal handling,
	// so we can't easily test it in unit tests without mocking.
	t.Skip("promptPassword requires stdin/terminal mocking")
}

func TestRunInit_MissingRepo(t *testing.T) {
	t.Parallel()

	// Clear environment variables that might be set
	originalRepo := os.Getenv("RESTIC_REPOSITORY")
	originalPass := os.Getenv("RESTIC_PASSWORD")
	os.Unsetenv("RESTIC_REPOSITORY")
	os.Unsetenv("RESTIC_PASSWORD")
	defer func() {
		if originalRepo != "" {
			os.Setenv("RESTIC_REPOSITORY", originalRepo)
		}
		if originalPass != "" {
			os.Setenv("RESTIC_PASSWORD", originalPass)
		}
	}()

	// Reset flags
	initRepo = ""
	initPassword = "testpass"
	initForce = false
	initDryRun = false

	// This should fail because repo is required and stdin is not a terminal
	err := runInit(nil, nil)
	if err == nil {
		t.Error("runInit() expected error when repo is missing, got nil")
	}
}

func TestRunInit_MissingPassword(t *testing.T) {
	t.Parallel()

	// Clear environment variables
	originalRepo := os.Getenv("RESTIC_REPOSITORY")
	originalPass := os.Getenv("RESTIC_PASSWORD")
	os.Unsetenv("RESTIC_REPOSITORY")
	os.Unsetenv("RESTIC_PASSWORD")
	defer func() {
		if originalRepo != "" {
			os.Setenv("RESTIC_REPOSITORY", originalRepo)
		}
		if originalPass != "" {
			os.Setenv("RESTIC_PASSWORD", originalPass)
		}
	}()

	// Set repo via flag but no password
	initRepo = "rest:https://test.example.com:8000"
	initPassword = ""
	initForce = false
	initDryRun = false

	// This should fail because password is required and stdin is not a terminal
	err := runInit(nil, nil)
	if err == nil {
		t.Error("runInit() expected error when password is missing, got nil")
	}
}
