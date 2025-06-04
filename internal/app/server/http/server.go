// Package server provides functionality to initialize and configure the
// HTTP server, including routes for handling URL shortening, user actions,
// and related API endpoints. The package uses the chi router and various
// middlewares for logging, JWT authentication, and gzip compression.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/handler"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
)

// Init initializes and returns a configured HTTP router with various
// routes and middlewares applied. The router is set up to handle different
// HTTP methods for URL shortening operations, including GET, POST, and DELETE.
//
// The router also includes middleware for logging, JWT authentication,
// and optional gzip compression for both request and response handling.
//
// Parameters:
//   - baseURL: The base URL for the shortened links.
//   - logger: A logger instance (typically used for logging requests and errors).
//   - withGzip: A flag indicating whether gzip compression should be enabled.
//   - sv: The service layer that handles URL shortening operations (implementing service.URLServiceIface).
//
// Returns:
//   - A chi router instance configured with the defined routes and middlewares.
func Init(baseURL string, trustedSubnet string, logger *zap.Logger, withGzip bool, sv service.URLServiceIface) *chi.Mux {

	// Create handler instances for different HTTP actions
	get := handler.NewGet(sv, logger)
	delete := handler.NewDelete(sv, logger)
	post := handler.NewPost(baseURL, sv, logger)

	// Create a new router
	r := chi.NewRouter()

	// Set allowed content types for incoming requests
	r.Use(chiMiddleware.AllowContentType("text/plain", "application/json", "text/html", "application/x-gzip"))

	// Use middleware for logging, JWT authentication, and optional gzip support
	r.Use(middleware.WithRequestLogging(logger))
	r.Use(middleware.WithJWT(service.NewAuth(sv)))

	// Enable gzip compression middleware if specified
	if withGzip {
		r.Use(middleware.WithGZIPPost)
		r.Use(middleware.WithGZIPGet)
	}

	// Define route handlers
	r.Post("/", post.PlainBody)                    // Handles POST requests for URL shortening
	r.Get("/{url}", get.ByShort)                   // Retrieves the original URL by shortened URL
	r.Get("/ping", get.PingDB)                     // Ping the database to check if it's accessible
	r.Get("/api/user/urls", get.URLsByUserID)      // Retrieve all URLs by the current user ID
	r.Delete("/api/user/urls", delete.DeleteBatch) // Delete a batch of URLs for the current user

	// Define routes for API-based URL shortening
	r.Route("/api/shorten", func(r chi.Router) {
		r.Post("/", post.HandlePostJSON)   // Handles POST requests with JSON payload
		r.Post("/batch", post.HandleBatch) // Handles batch URL shortening requests
	})

	// Route for requesting shortened statistics.
	r.Route("/api/internal", func(r chi.Router) {
		r.Use(middleware.WithSubnet(trustedSubnet))
		r.Get("/stats", get.Stats)
	})

	// Default route if no shortened URL is provided
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Short URL is required", http.StatusBadRequest)
	})

	// Handler for unsupported HTTP methods
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	// Handler for routes not found
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Route not found", http.StatusNotFound)
	})

	return r
}
