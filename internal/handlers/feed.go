package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/httputil"
	mw "github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/middleware"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/models"
	"github.com/NguyenDuyHieu11/rewrite_social_media_app/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FeedHandler exposes home-feed write endpoints on the dispatcher. The parent
// router must mount mw.RequireAuth before this sub-router; handlers assume
// claims are present.
type FeedHandler struct {
	svc *services.FeedService
}

func NewFeedHandler(svc *services.FeedService) *FeedHandler {
	return &FeedHandler{svc: svc}
}

// Routes returns a chi.Router meant to be mounted at /posts.
func (h *FeedHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.CreatePost)
	r.Get("/{postID}", h.GetPost)
	r.Post("/{postID}/comments", h.CreateComment)
	r.Put("/{postID}/reactions", h.UpsertReaction)
	return r
}

type createPostRequest struct {
	Body string `json:"body"`
}

type createCommentRequest struct {
	Body string `json:"body"`
}

type upsertReactionRequest struct {
	Type string `json:"type"`
}

func (h *FeedHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r.Context())
	authorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "token subject is not a valid user id")
		return
	}

	var req createPostRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "body must not be empty")
		return
	}

	post, err := h.svc.CreatePost(r.Context(), authorID, req.Body)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create post")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, post)
}

func (h *FeedHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_post_id", "postID must be a UUID")
		return
	}

	post, err := h.svc.GetPost(r.Context(), postID)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "post not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get post")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, post)
}

func (h *FeedHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r.Context())
	authorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "token subject is not a valid user id")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_post_id", "postID must be a UUID")
		return
	}

	var req createCommentRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", "body must not be empty")
		return
	}

	comment, err := h.svc.CreateComment(r.Context(), postID, authorID, req.Body)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "post not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create comment")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, comment)
}

func (h *FeedHandler) UpsertReaction(w http.ResponseWriter, r *http.Request) {
	claims := mw.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "token subject is not a valid user id")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_post_id", "postID must be a UUID")
		return
	}

	var req upsertReactionRequest
	if err := httputil.DecodeJSON(r, &req, 0); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	typ := models.ReactionType(req.Type)
	if !models.ValidReactionType(typ) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_type", "type must be one of like, love, haha, sad, angry")
		return
	}

	reaction, err := h.svc.UpsertReaction(r.Context(), postID, userID, typ)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "post not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to upsert reaction")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, reaction)
}
