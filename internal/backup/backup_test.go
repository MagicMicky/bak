package backup

import (
	"encoding/json"
	"testing"

	"github.com/magicmicky/bak/internal/config"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Tag:   "test",
		Paths: []string{"/tmp"},
	}

	runner := NewRunner(cfg)

	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}
	if runner.Config != cfg {
		t.Error("Config not set correctly")
	}
	if runner.Verbose {
		t.Error("Verbose should default to false")
	}
}

func TestSnapshot_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"short_id": "abc123",
		"id": "abc123def456",
		"time": "2024-01-15T03:45:00.123456789Z",
		"hostname": "server1",
		"paths": ["/var/www", "/etc/nginx"],
		"tags": ["webapp", "retain:h=0,d=7,w=4,m=6,y=0"]
	}`

	var snapshot Snapshot
	err := json.Unmarshal([]byte(jsonData), &snapshot)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if snapshot.ShortID != "abc123" {
		t.Errorf("ShortID = %q, want %q", snapshot.ShortID, "abc123")
	}
	if snapshot.ID != "abc123def456" {
		t.Errorf("ID = %q, want %q", snapshot.ID, "abc123def456")
	}
	if snapshot.Hostname != "server1" {
		t.Errorf("Hostname = %q, want %q", snapshot.Hostname, "server1")
	}
	if len(snapshot.Paths) != 2 {
		t.Errorf("len(Paths) = %d, want 2", len(snapshot.Paths))
	}
	if len(snapshot.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(snapshot.Tags))
	}
}

func TestSnapshot_UnmarshalJSONArray(t *testing.T) {
	t.Parallel()

	jsonData := `[
		{"short_id": "abc123", "time": "2024-01-15T03:45:00Z", "paths": ["/var/www"]},
		{"short_id": "def456", "time": "2024-01-14T03:42:00Z", "paths": ["/opt/app"]}
	]`

	var snapshots []Snapshot
	err := json.Unmarshal([]byte(jsonData), &snapshots)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("len(snapshots) = %d, want 2", len(snapshots))
	}
	if snapshots[0].ShortID != "abc123" {
		t.Errorf("snapshots[0].ShortID = %q, want %q", snapshots[0].ShortID, "abc123")
	}
	if snapshots[1].ShortID != "def456" {
		t.Errorf("snapshots[1].ShortID = %q, want %q", snapshots[1].ShortID, "def456")
	}
}
