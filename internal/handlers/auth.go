// Package handlers holds HTTP handlers that wrap the service layer.
//
// Handlers are intentionally thin: they decode the request, call into a
// service, and translate the service's typed result (or sentinel error)
// into an HTTP response via internal/httputil. They never do business logic.
package handlers

import (
	"errors"
	"net/http"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/httputil"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/services"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Routes returns a chi.Router with the four auth endpoints mounted. Mount
// with `r.Mount("/auth", h.Routes())` from the parent router.
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	return r
}

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// authResponse is shared by register, login, and refresh.
type authResponse struct {
	User         any    `json:"user,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"refresh_expires_at"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "username, email, password are required")
		return
	}

	user, pair, err := h.svc.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, authResponse{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Username == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "username and password are required")
		return
	}

	user, pair, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, authResponse{
		User:         user,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.RefreshToken == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "refresh_token is required")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, authResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.RefreshExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.RefreshToken == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "refresh_token is required")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeServiceError centralises the service-error -> HTTP-status mapping.
// New sentinel errors get one new case here; nothing else changes.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, services.ErrTokenReuse):
		httputil.WriteError(w, http.StatusUnauthorized, "token_reuse", "refresh token reuse detected; all sessions revoked")
	default:
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
