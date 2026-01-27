package notify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Send(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var receivedTitle, receivedBody string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
				}

				body, _ := io.ReadAll(r.Body)
				var p payload
				json.Unmarshal(body, &p)
				receivedTitle = p.Title
				receivedBody = p.Body

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := New(server.URL)
			err := client.Send("Test Title", "Test Body")

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if receivedTitle != "Test Title" {
					t.Errorf("title = %q, want %q", receivedTitle, "Test Title")
				}
				if receivedBody != "Test Body" {
					t.Errorf("body = %q, want %q", receivedBody, "Test Body")
				}
			}
		})
	}
}

func TestClient_SendSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tag        string
		snapshotID string
		wantTitle  string
		wantBody   string
	}{
		{
			name:       "with snapshot ID",
			tag:        "webapp",
			snapshotID: "abc123",
			wantTitle:  "Backup completed: webapp",
			wantBody:   "Backup 'webapp' completed successfully.\nSnapshot: abc123",
		},
		{
			name:       "without snapshot ID",
			tag:        "myapp",
			snapshotID: "",
			wantTitle:  "Backup completed: myapp",
			wantBody:   "Backup 'myapp' completed successfully.",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var receivedTitle, receivedBody string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var p payload
				json.Unmarshal(body, &p)
				receivedTitle = p.Title
				receivedBody = p.Body
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := New(server.URL)
			err := client.SendSuccess(tt.tag, tt.snapshotID)

			if err != nil {
				t.Errorf("SendSuccess() error = %v", err)
			}

			if receivedTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", receivedTitle, tt.wantTitle)
			}
			if receivedBody != tt.wantBody {
				t.Errorf("body = %q, want %q", receivedBody, tt.wantBody)
			}
		})
	}
}

func TestClient_SendFailure(t *testing.T) {
	t.Parallel()

	var receivedTitle, receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p payload
		json.Unmarshal(body, &p)
		receivedTitle = p.Title
		receivedBody = p.Body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL)
	backupErr := errors.New("connection refused")
	err := client.SendFailure("webapp", backupErr)

	if err != nil {
		t.Errorf("SendFailure() error = %v", err)
	}

	wantTitle := "Backup failed: webapp"
	wantBody := "Backup 'webapp' failed.\nError: connection refused"

	if receivedTitle != wantTitle {
		t.Errorf("title = %q, want %q", receivedTitle, wantTitle)
	}
	if receivedBody != wantBody {
		t.Errorf("body = %q, want %q", receivedBody, wantBody)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	client := New("http://example.com/notify")

	if client.URL != "http://example.com/notify" {
		t.Errorf("URL = %q, want %q", client.URL, "http://example.com/notify")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}
