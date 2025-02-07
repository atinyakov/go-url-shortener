package server

import (
	"database/sql"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/handlers"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func Init(resolver *services.URLResolver, baseURL string, logger logger.LoggerI, withGzip bool, fs storage.StorageI, db *sql.DB) *chi.Mux {

	getHandler := handlers.NewGetHandler(resolver, fs, logger, db)
	postHandler := handlers.NewPostHandler(resolver, baseURL, fs, logger, db)

	r := chi.NewRouter()
	r.Use(chiMiddleware.AllowContentType("text/plain", "application/json", "text/html", "application/x-gzip"))
	r.Use(middleware.WithRequestLogging(logger))

	if withGzip {
		r.Use(middleware.WithGZIPPost)
		r.Use(middleware.WithGZIPGet)
	}

	r.Post("/", postHandler.HandlePostPlainBody)
	r.Post("/api/shorten", postHandler.HandlePostJSON)
	r.Get("/{url}", getHandler.HandleGet)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Short URL is required", http.StatusBadRequest)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Route not found", http.StatusNotFound)
	})

	return r
}
