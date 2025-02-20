package server

import (
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/handler"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func Init(baseURL string, logger *zap.Logger, withGzip bool, sv *service.URLService) *chi.Mux {
	authService := service.NewAuth(sv)

	get := handler.NewGet(sv, logger, authService)
	post := handler.NewPost(baseURL, sv, logger, authService)

	r := chi.NewRouter()
	r.Use(chiMiddleware.AllowContentType("text/plain", "application/json", "text/html", "application/x-gzip"))
	r.Use(middleware.WithRequestLogging(logger))
	r.Use(middleware.WithJWT(service.NewAuth(sv)))

	if withGzip {
		r.Use(middleware.WithGZIPPost)
		r.Use(middleware.WithGZIPGet)
	}

	r.Post("/", post.PlainBody)
	r.Get("/{url}", get.ByShort)
	r.Get("/ping", get.PingDB)
	r.Get("/api/user/urls", get.URLsByUserID)

	r.Route("/api/shorten", func(r chi.Router) {
		r.Post("/", post.HandlePostJSON)
		r.Post("/batch", post.HandleBatch)
	})

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
