package handlers

import (
	"context"
	"database/sql"
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
	db       *sql.DB
}

func NewGetHandler(resolver *services.URLResolver, s storage.StorageI, l logger.LoggerI, db *sql.DB) *GetHandler {
	return &GetHandler{
		Resolver: resolver,
		storage:  s,
		logger:   l,
		db:       db,
	}
}

// HandleGet handles GET requests for URL resolution
func (h *GetHandler) HandleGet(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "url")

	fmt.Println("sorturl", shortURL)

	longURL := h.Resolver.ShortToLong(shortURL)
	if longURL == "" {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	res.Header().Set("Location", longURL)

	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *GetHandler) HandlePing(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.db.PingContext(ctx); err != nil {
		http.Error(res, "db ping failed", http.StatusInternalServerError)
	}

	res.WriteHeader(http.StatusOK)
}
