package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
	maxRetries     = 3
	baseRetryDelay = 500 * time.Millisecond
)

// Payload is the JSON body accepted by Apprise API /notify/ endpoints.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Type  string `json:"type,omitempty"`
}

// Notify POSTs payload to the Apprise API URL with retries. Returns nil on success.
func Notify(client *http.Client, url string, payload Payload) error {
	if url == "" {
		return nil
	}
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	if payload.Type == "" {
		payload.Type = "info"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal apprise payload: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(baseRetryDelay * (1 << (attempt - 1)))
		}
		if err := postOnce(client, url, body); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("apprise notify failed after %d attempts: %w", maxRetries, lastErr)
}

func postOnce(client *http.Client, url string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// FormatOrderBody builds a human-readable Apprise body with a bulleted item list.
func FormatOrderBody(orderID string, itemDescriptions []string, username string) string {
	var lines []string
	if username != "" {
		lines = append(lines, fmt.Sprintf("Ordered by %s", username))
	}
	if orderID != "" {
		short := orderID
		if len(short) > 8 {
			short = short[:8]
		}
		lines = append(lines, fmt.Sprintf("Order #%s", short))
	}
	for _, item := range itemDescriptions {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		lines = append(lines, "• "+item)
	}
	return strings.Join(lines, "\n")
}
