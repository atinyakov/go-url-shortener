// Package handler provides HTTP handlers for resolving shortened URLs and fetching data related to URLs.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
)

// GetHandler handles GET requests related to URL resolution and user-specific URLs.
type GetHandler struct {
	service service.URLServiceIface // Service for handling URL operations.
	logger  *zap.Logger             // Logger for logging events.
}

// NewGet creates a new GetHandler instance with provided URL service and logger.
func NewGet(s service.URLServiceIface, l *zap.Logger) *GetHandler {
	return &GetHandler{
		service: s,
		logger:  l,
	}
}

// ByShort handles GET requests for URL resolution using a shortened URL.
// It returns a 302 redirect to the original URL if found, or a 404 error if not found.
func (h *GetHandler) ByShort(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract the shortened URL from the request parameters.
	shortURL := chi.URLParam(req, "url")
	h.logger.Info("Got URL from request params:", zap.String("shortURL", shortURL))

	// Resolve the original URL using the service.
	r, err := h.service.GetURLByShort(ctx, shortURL)
	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	// Check if the URL is marked as deleted.
	if r.IsDeleted {
		res.WriteHeader(http.StatusGone)
	}

	// Set the Location header to the original URL and send a temporary redirect response.
	res.Header().Set("Location", r.Original)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

// PingDB handles GET requests for checking the health of the database connection.
// It returns a 200 status if the database is reachable, or 500 if there is an error.
func (h *GetHandler) PingDB(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Check the database connection status.
	if err := h.service.PingContext(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// URLsByUserID handles GET requests for retrieving all URLs associated with a specific user.
// It returns a list of URLs in JSON format or a 204 No Content status if no URLs are found.
func (h *GetHandler) URLsByUserID(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract user ID from request context.
	val := req.Context().Value(middleware.UserIDKey)
	userID, ok := val.(string)
	if !ok {
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Retrieve the URLs associated with the user from the service.
	urls, err := h.service.GetURLByUserID(ctx, userID)
	if err != nil {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	// If no URLs are found, return a 204 No Content status.
	if len(*urls) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	// Marshal the list of URLs to JSON and send the response.
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
