package server

import (
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/handlers"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/go-chi/chi/v5"
)

func Init(resolver *services.URLResolver, baseURL string) *chi.Mux {

	handler := handlers.NewURLHandler(resolver, baseURL)

	r := chi.NewRouter()

	r.Post("/", handler.HandlePost)
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
