package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
)

type GetHandler struct {
	service service.URLServiceIface
	logger  *zap.Logger
}

func NewGet(s service.URLServiceIface, l *zap.Logger) *GetHandler {
	return &GetHandler{
		service: s,
		logger:  l,
	}
}

// ByShort handles GET requests for URL resolution
func (h *GetHandler) ByShort(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	shortURL := chi.URLParam(req, "url")
	h.logger.Info("Got URL from request params:", zap.String("shortURL", shortURL))

	r, err := h.service.GetURLByShort(ctx, shortURL)
	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	if r.IsDeleted {
		res.WriteHeader(http.StatusGone)
	}

	res.Header().Set("Location", r.Original)

	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *GetHandler) PingDB(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()
	if err := h.service.PingContext(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (h *GetHandler) URLsByUserID(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	val := req.Context().Value(middleware.UserIDKey)
	userID, ok := val.(string)
	if !ok {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetURLByUserID(ctx, userID)

	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	if len(*urls) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	response, err := json.Marshal(*urls)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}
