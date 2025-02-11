package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

type GetHandler struct {
	Resolver *services.URLResolver
	storage  storage.StorageI
	logger   logger.LoggerI
	// db       *sql.DB
}

func NewGetHandler(resolver *services.URLResolver, s storage.StorageI, l logger.LoggerI) *GetHandler {
	return &GetHandler{
		Resolver: resolver,
		storage:  s,
		logger:   l,
		// db:       db,
	}
}

// HandleGet handles GET requests for URL resolution
func (h *GetHandler) HandleGet(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "url")

	fmt.Println("short url:", shortURL)

	longURL := h.Resolver.ShortToLong(shortURL)
	if longURL == "" {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}
	fmt.Println("Found LONG url:", longURL)

	res.Header().Set("Location", longURL)

	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *GetHandler) HandlePing(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	if err := h.storage.PingContext(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	res.WriteHeader(http.StatusOK)
}
