package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type GetHandler struct {
	service *service.URLService
	logger  *zap.Logger
	auth    *service.Auth
}

func NewGet(s *service.URLService, l *zap.Logger, auth *service.Auth) *GetHandler {
	return &GetHandler{
		service: s,
		logger:  l,
		auth:    auth,
	}
}

// ByShort handles GET requests for URL resolution
func (h *GetHandler) ByShort(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "url")
	h.logger.Info("Got URL from request params:", zap.String("shortURL", shortURL))

	r, err := h.service.GetURLByShort(shortURL)
	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
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

	userID := req.Context().Value(middleware.UserIDKey).(string)
	if userID == "" {
		http.Error(res, "", http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetURLByUserID(userID)

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
