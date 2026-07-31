// Package events defines the wire format for realtime messages on the pub/sub
// bus. The dispatcher marshals domain objects into an Envelope before
// publishing; the gateway splits the envelope back into an SSE event type and
// data payload. Keeping both directions in one package is what stops the two
// binaries from drifting apart.
package events

import (
	"encoding/json"
	"fmt"
)

// Event types carried in Envelope.Type. The gateway forwards these verbatim
// as the SSE "event:" field, so browser clients can addEventListener on them.
const (
	TypeComment  = "comment"
	TypeReaction = "reaction"
)

// Envelope wraps one domain event for transport over the bus.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Marshal encodes data inside an Envelope of the given event type.
func Marshal(eventType string, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("events: marshal %s data: %w", eventType, err)
	}
	payload, err := json.Marshal(Envelope{Type: eventType, Data: raw})
	if err != nil {
		return nil, fmt.Errorf("events: marshal %s envelope: %w", eventType, err)
	}
	return payload, nil
}

// Split maps a bus payload to an SSE event type and data bytes. Payloads that
// are not a valid Envelope fall back to the SSE default event type "message"
// with the payload passed through untouched, so a malformed publisher can
// never wedge the delivery loop.
func Split(payload []byte) (eventType string, data []byte) {
	var env Envelope
	if err := json.Unmarshal(payload, &env); err == nil && env.Type != "" && len(env.Data) > 0 {
		return env.Type, env.Data
	}
	return "message", payload
}
