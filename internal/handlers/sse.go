package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/events"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/feed"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/httputil"
	mw "github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/middleware"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/sse"
	"github.com/go-chi/chi/v5"
)

// SSEHandler serves long-lived Server-Sent Event streams for post subscriptions.
// It depends on feed.Hub for Redis + in-memory wiring. Publishing is the
// dispatcher's job; this handler only delivers.
type SSEHandler struct {
	hub *feed.Hub
	log *slog.Logger
}

// NewSSEHandler constructs an SSEHandler. hub and log must be non-nil.
func NewSSEHandler(hub *feed.Hub, log *slog.Logger) *SSEHandler {
	return &SSEHandler{hub: hub, log: log}
}

// Routes returns a chi router mounted at /sse by the gateway main.
//
// Endpoints:
//
//	PUT /sse/posts/{postID}/subscriptions
//
// Requires authentication. Bearer or ?access_token= both work: the gateway
// mounts mw.QueryTokenAsBearer before mw.Auth, so by the time this router
// runs, claims are already on the context and RequireAuth has passed.
func (h *SSEHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Put("/posts/{postID}/subscriptions", h.SubscribePost)
	return r
}

// SubscribePost opens an SSE stream for one post.
//
// Request contract:
//   - Method PUT (per project spec; idempotent "subscribe" semantics).
//   - Path param postID — which post to watch.
//   - Header Accept must include text/event-stream.
//   - Valid JWT on context (via Auth + RequireAuth middleware).
//
// Response:
//   - 200 with Content-Type text/event-stream; body never ends until disconnect.
//   - First event: event: ready, data: {"status":"connected"}.
//   - Comment heartbeats every 25s (: keepalive) to defeat proxy timeouts.
//   - Further events from Redis via Hub → splitEvent → sse.WriteEvent.
//
// Lifecycle:
//  1. hub.Acquire — may start Redis SUBSCRIBE + bridge for this post.
//  2. Loop: client disconnect, keepalive tick, or event on local channel.
//  3. defer hub.Release — may UNSUBSCRIBE Redis when last viewer leaves.
//
// The gateway HTTP server must use WriteTimeout=0 for this handler to work.
func (h *SSEHandler) SubscribePost(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r.Context())
	if claims == nil {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	postID := chi.URLParam(r, "postID")
	if postID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "postID is required")
		return
	}

	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		httputil.WriteError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must include text/event-stream")
		return
	}

	local, connID, err := h.hub.Acquire(r.Context(), postID, claims.UserID)
	if err != nil {
		h.log.Error("sse acquire failed", "post_id", postID, "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to subscribe")
		return
	}
	defer h.hub.Release(r.Context(), postID, claims.UserID, connID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if present

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "streaming not supported")
		return
	}
	flusher.Flush()

	_ = sse.WriteEvent(w, "ready", []byte(`{"status":"connected"}`))

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-keepAlive.C:
			// SSE comment lines are ignored by EventSource parsers but keep
			// the TCP connection warm through proxies and load balancers.
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			flusher.Flush()

		case payload, ok := <-local:
			if !ok {
				return
			}
			// Bus payloads are events.Envelope JSON published by the
			// dispatcher; Split recovers the SSE event type and data.
			eventType, data := events.Split(payload)
			if err := sse.WriteEvent(w, eventType, data); err != nil {
				return
			}
		}
	}
}
