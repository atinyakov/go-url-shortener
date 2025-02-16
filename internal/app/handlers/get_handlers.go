package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
)

type GetHandler struct {
	Resolver *services.URLResolver
	service  *services.URLService
	logger   *logger.Logger
}

func NewGetHandler(resolver *services.URLResolver, s *services.URLService, l *logger.Logger) *GetHandler {
	return &GetHandler{
		Resolver: resolver,
		service:  s,
		logger:   l,
	}
}

// HandleGet handles GET requests for URL resolution
func (h *GetHandler) HandleGet(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "url")
	h.logger.Info(fmt.Sprintf("Got URL from request params: %s", shortURL))

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
