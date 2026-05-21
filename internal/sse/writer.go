// Package sse provides helpers for writing Server-Sent Events (SSE) frames to
// an http.ResponseWriter.
//
// SSE is the transport for the home-feed domain on the gateway: one long-lived
// HTTP response streams many small events to the browser. This package only
// handles wire formatting and flushing — it does not manage subscriptions,
// Redis, or auth (see internal/handlers/sse.go and internal/feed/hub.go).
//
// Clients (EventSource) expect:
//   - Content-Type: text/event-stream (set by the handler)
//   - Each event terminated by a blank line
//   - Proxies may buffer unless the handler flushes after each event
//
// Handlers may also send comment lines (": keepalive\n\n") for heartbeats;
// those are written directly on the ResponseWriter, not through this package.
package sse

import (
	"bufio"
	"fmt"
	"net/http"
)

// WriteEvent writes a single SSE event and flushes it to the client immediately.
//
// Parameters:
//   - eventType — value for the "event:" field (e.g. "comment", "ready").
//     If empty, the event line is omitted and the client uses the default
//     event type "message".
//   - data — raw bytes for the "data:" field. Must not contain raw newlines;
//     multi-line SSE data requires multiple "data:" lines (not supported here).
//
// Wire format produced:
//
//	event: comment
//	data: {"id":"..."}
//
// (blank line ends the event)
//
// The ResponseWriter must implement http.Flusher (chi/net/http default
// wrappers do). Returns an error if flushing is unsupported or I/O fails.
//
// Callers should set SSE headers before the first WriteEvent (see
// handlers.SSEHandler.SubscribePost).
func WriteEvent(w http.ResponseWriter, eventType string, data []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("sse: ResponseWriter does not support Flush")
	}

	bw := bufio.NewWriter(w)

	if eventType != "" {
		if _, err := fmt.Fprintf(bw, "event: %s\n", eventType); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(bw, "data: %s\n\n", data); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
