package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type GetHandler struct {
	service *service.URLService
	logger  *zap.Logger
}

func NewGet(s *service.URLService, l *zap.Logger) *GetHandler {
	return &GetHandler{
		service: s,
		logger:  l,
	}
}

// HandleGet handles GET requests for URL resolution
func (h *GetHandler) HandleGet(res http.ResponseWriter, req *http.Request) {
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

func (h *GetHandler) HandlePing(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()
	if err := h.service.PingContext(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}
