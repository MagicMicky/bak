// Package notify handles sending notifications via Apprise HTTP API.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client sends notifications to an Apprise endpoint.
type Client struct {
	URL        string
	HTTPClient *http.Client
}

// payload is the JSON body sent to Apprise.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// New creates a new notification client.
func New(url string) *Client {
	return &Client{
		URL: url,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a notification with the given title and message.
func (c *Client) Send(title, message string) error {
	p := payload{
		Title: title,
		Body:  message,
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification failed with status %d", resp.StatusCode)
	}

	return nil
}

// SendSuccess sends a success notification for a completed backup.
func (c *Client) SendSuccess(tag, snapshotID string) error {
	title := fmt.Sprintf("Backup completed: %s", tag)
	message := fmt.Sprintf("Backup '%s' completed successfully.", tag)
	if snapshotID != "" {
		message += fmt.Sprintf("\nSnapshot: %s", snapshotID)
	}
	return c.Send(title, message)
}

// SendFailure sends a failure notification for a failed backup.
func (c *Client) SendFailure(tag string, backupErr error) error {
	title := fmt.Sprintf("Backup failed: %s", tag)
	message := fmt.Sprintf("Backup '%s' failed.\nError: %v", tag, backupErr)
	return c.Send(title, message)
}
