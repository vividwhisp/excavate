package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// EventType mirrors research pipeline lifecycle.
type EventType string

const (
	EventProgress EventType = "progress"
	EventSources  EventType = "sources"
	EventDelta    EventType = "delta"
	EventDone     EventType = "done"
	EventError    EventType = "error"
)

type Event struct {
	Type    EventType       `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewEvent(t EventType, v any) (Event, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return Event{}, err
	}
	return Event{Type: t, Payload: b}, nil
}

func Channel(messageID string) string { return "stream:" + messageID }

// WritePreamble sends SSE headers and the initial comment so the connection
// is established before any research work begins.
func WritePreamble(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, ": connected\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// WriteEvent writes a single SSE frame and flushes.
func WriteEvent(w http.ResponseWriter, ev Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

// Forward pumps events from a Redis pub/sub channel to the HTTP response
// until ctx is done, the client disconnects, or a terminal (done/error) event
// is received.
func Forward(ctx context.Context, w http.ResponseWriter, r *http.Request, rdb *redis.Client, channel string) error {
	sub := rdb.Subscribe(ctx, channel)
	defer sub.Close()

	// Wait for the SUBSCRIBE to be acknowledged so we don't miss the early
	// events that arrive milliseconds after the client connects.
	if err := sub.Ping(ctx); err != nil {
		return fmt.Errorf("subscribe %s: %w", channel, err)
	}

	ch := sub.Channel()
	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	disconnected := r.Context().Done()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-disconnected:
			return nil
		case <-keepAlive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			// msg.Payload is already the serialized Event JSON from Publish.
			var ev Event
			if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil {
				if ev.Type == EventDone || ev.Type == EventError {
					return WriteEvent(w, ev)
				}
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}

// Publish sends an event to a message's channel.
func Publish(ctx context.Context, rdb *redis.Client, messageID string, ev Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return rdb.Publish(ctx, Channel(messageID), b).Err()
}
