package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrinter_ColorOutput(t *testing.T) {
	t.Parallel()

	// Ensure NO_COLOR is not set for this test
	os.Unsetenv("NO_COLOR")

	tests := []struct {
		name     string
		method   func(*Printer, string, ...any)
		expected string
	}{
		{
			name:     "Header",
			method:   (*Printer).Header,
			expected: cyan + bold + "test message" + reset + "\n",
		},
		{
			name:     "Success",
			method:   (*Printer).Success,
			expected: green + "test message" + reset + "\n",
		},
		{
			name:     "Warning",
			method:   (*Printer).Warning,
			expected: yellow + "test message" + reset + "\n",
		},
		{
			name:     "Error",
			method:   (*Printer).Error,
			expected: red + "test message" + reset + "\n",
		},
		{
			name:     "Info",
			method:   (*Printer).Info,
			expected: "test message\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := New(&buf)
			tt.method(p, "test message")

			if got := buf.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrinter_NoColorEnv(t *testing.T) {
	// Set NO_COLOR environment variable
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	var buf bytes.Buffer
	p := New(&buf)

	p.Header("header")
	p.Success("success")
	p.Warning("warning")
	p.Error("error")

	output := buf.String()

	// Should not contain any ANSI codes
	if strings.Contains(output, "\033[") {
		t.Errorf("output contains ANSI codes when NO_COLOR is set: %q", output)
	}

	// Should still have the messages
	expectedLines := []string{"header", "success", "warning", "error"}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("output missing %q", line)
		}
	}
}

func TestPrinter_FormatArgs(t *testing.T) {
	t.Parallel()

	os.Unsetenv("NO_COLOR")

	var buf bytes.Buffer
	p := New(&buf)
	p.noColor = true // Force no color for easier testing

	p.Info("count: %d, name: %s", 42, "test")

	expected := "count: 42, name: test\n"
	if got := buf.String(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestDefault(t *testing.T) {
	t.Parallel()

	p := Default()
	if p == nil {
		t.Fatal("Default() returned nil")
	}
	if p.out != os.Stdout {
		t.Error("Default() should write to stdout")
	}
}
