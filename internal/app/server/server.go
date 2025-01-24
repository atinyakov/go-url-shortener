package server

import (
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/handlers"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func Init(resolver *services.URLResolver, baseURL string, logger logger.LoggerI, withGzip bool) *chi.Mux {

	handler := handlers.NewURLHandler(resolver, baseURL)

	r := chi.NewRouter()
	r.Use(chiMiddleware.AllowContentType("text/plain", "application/json", "text/html", "application/x-gzip"))
	r.Use(middleware.WithLogging(logger))

	if withGzip {
		r.Use(middleware.WithGZIP)
	}

	r.Post("/", handler.HandlePostPlainBody)
	r.Post("/api/shorten", handler.HandlePostJSON)
	r.Get("/{url}", handler.HandleGet)

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
