package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const sseKeepaliveInterval = 30 * time.Second

// orderEvent is sent to SSE clients on status update.
type orderEvent struct {
	Type    string `json:"type"`
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// sseBroadcaster sends order status updates to all connected SSE clients.
type sseBroadcaster struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newSSEBroadcaster() *sseBroadcaster {
	return &sseBroadcaster{clients: make(map[chan []byte]struct{})}
}

func (b *sseBroadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 8)
	b.clients[ch] = struct{}{}
	return ch
}

func (b *sseBroadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

// Broadcast drops the message for clients whose buffer is full rather than blocking the publisher.
func (b *sseBroadcaster) Broadcast(data []byte) {
	b.mu.Lock()
	clients := make([]chan []byte, 0, len(b.clients))
	for ch := range b.clients {
		clients = append(clients, ch)
	}
	b.mu.Unlock()
	for _, ch := range clients {
		select {
		case ch <- data:
		default:
		}
	}
}

func (b *sseBroadcaster) BroadcastOrderEvent(ev orderEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		logrus.Warnf("Marshal order event: %v", err)
		return
	}
	b.Broadcast(data)
}

// ServeHTTP serves GET /orders/events as text/event-stream, with keepalives every sseKeepaliveInterval.
func (b *sseBroadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, err := prepareEventStream(w, r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)
	streamEvents(r.Context(), w, flusher, ch)
}

func prepareEventStream(w http.ResponseWriter, r *http.Request) (http.Flusher, error) {
	if r.Method != http.MethodGet {
		return nil, errMethodNotAllowed
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, &httpError{http.StatusInternalServerError, "streaming unsupported"}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return flusher, nil
}

func streamEvents(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, ch <-chan []byte) {
	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()
	for {
		msg, ok := nextEventMessage(ctx, ch, keepalive.C)
		if !ok {
			return
		}
		if _, err := w.Write(msg); err != nil {
			return
		}
		flusher.Flush()
	}
}

func nextEventMessage(ctx context.Context, ch <-chan []byte, keepalive <-chan time.Time) ([]byte, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case data, ok := <-ch:
		return append(append([]byte("data: "), data...), '\n', '\n'), ok
	case <-keepalive:
		return []byte(": keepalive\n\n"), true
	}
}
