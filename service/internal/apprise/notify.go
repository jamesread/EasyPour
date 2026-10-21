package apprise

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"easypour/service/internal/httpjson"
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
	body, err := json.Marshal(withDefaultType(payload))
	if err != nil {
		return fmt.Errorf("marshal apprise payload: %w", err)
	}
	if err := httpjson.PostWithRetry(clientOrDefault(client), url, body); err != nil {
		return fmt.Errorf("apprise notify: %w", err)
	}
	return nil
}

func withDefaultType(payload Payload) Payload {
	if payload.Type == "" {
		payload.Type = "info"
	}
	return payload
}

func clientOrDefault(client *http.Client) *http.Client {
	if client == nil {
		return &http.Client{Timeout: httpjson.DefaultTimeout}
	}
	return client
}

// FormatOrderBody builds a human-readable Apprise body with a bulleted item list.
func FormatOrderBody(orderID string, itemDescriptions []string, username string) string {
	lines := orderHeaderLines(orderID, username)
	return strings.Join(append(lines, bulletLines(itemDescriptions)...), "\n")
}

func orderHeaderLines(orderID, username string) []string {
	var lines []string
	if username != "" {
		lines = append(lines, "Ordered by "+username)
	}
	if orderID != "" {
		lines = append(lines, "Order #"+shortOrderID(orderID))
	}
	return lines
}

func shortOrderID(orderID string) string {
	if len(orderID) > 8 {
		return orderID[:8]
	}
	return orderID
}

func bulletLines(items []string) []string {
	var lines []string
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			lines = append(lines, "• "+item)
		}
	}
	return lines
}
