package httpjson

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultTimeout bounds each individual POST attempt.
	DefaultTimeout = 15 * time.Second
	// MaxAttempts is the number of POST attempts before giving up.
	MaxAttempts    = 3
	baseRetryDelay = 500 * time.Millisecond
)

// PostWithRetry POSTs body as JSON to url, retrying failed attempts with exponential backoff.
func PostWithRetry(client *http.Client, url string, body []byte) error {
	var err error
	for attempt := range MaxAttempts {
		time.Sleep(backoffDelay(attempt))
		if err = postOnce(client, url, body); err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", MaxAttempts, err)
}

func backoffDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	return baseRetryDelay << (attempt - 1)
}

func postOnce(client *http.Client, url string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
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
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return statusError(resp.StatusCode)
}

func statusError(code int) error {
	if code < 200 || code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}
