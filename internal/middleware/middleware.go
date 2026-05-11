package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/auth"
	"github.com/google/uuid"
)

// contextKey is unexported so no other package can produce a colliding key.
// Using a typed key (rather than a bare string) is the idiomatic Go way to
// stash values on a context.
type contextKey string

const (
	contextKeyRequestID contextKey = "request_id"
	contextKeyClaims    contextKey = "claims"
)

// requestIDHeader is the HTTP header we both read (if the upstream caller
// already attached one — useful for tracing across services) and write back
// to the response so the client can correlate logs.
const requestIDHeader = "X-Request-Id"

// RequestID injects a unique request ID into the request context and
// surfaces it on the response. If the incoming request already has an
// X-Request-Id header, we honour it; otherwise we generate a UUID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), contextKeyRequestID, id)
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Context.Value()
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyRequestID).(string)
	return v
}

type statusRecorder struct {
	bytes  int
	status int
	http.ResponseWriter
}

// WriteHeader observes the status the handler chose. http.ResponseWriter
// doesn't expose the chosen code after the fact, so we have to intercept.
func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Write tracks bytes and defaults the status to 200 if the handler skipped
// WriteHeader and went straight to Write (Go's default).
func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func Logger(log func(r *http.Request, status, bytes int, dur time.Duration)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			log(r, rec.status, rec.bytes, time.Since(start))
		})
	}
}

// auth

func Auth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := auth.ParseAccessToken(token, secret)
			if err != nil {
				// Token was provided but invalid; refuse the request rather
				// than silently dropping the claim.
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth refuses any request that doesn't already have valid claims on
// its context. Mount it after Auth on routes that must have a logged-in user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(contextKeyClaims).(*auth.Claims); !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// ClaimsFromContext returns the *auth.Claims stashed by the Auth middleware,
// or nil if no valid token was on the request.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	v, _ := ctx.Value(contextKeyClaims).(*auth.Claims)
	return v
}
