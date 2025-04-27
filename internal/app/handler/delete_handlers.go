// Package handler provides HTTP handlers for managing URL deletion operations.
package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

// DeleteHandler handles HTTP requests for deleting URLs.
type DeleteHandler struct {
	service service.URLServiceIface // The service for URL-related operations.
	logger  *zap.Logger             // Logger for logging events.
}

// NewDelete creates a new instance of DeleteHandler with the provided URL service and logger.
func NewDelete(s service.URLServiceIface, l *zap.Logger) *DeleteHandler {
	return &DeleteHandler{
		service: s,
		logger:  l,
	}
}

// DeleteBatch handles DELETE requests for deleting multiple URLs in batch.
// It reads a list of shortened URLs from the request body and deletes them asynchronously.
// A 202 Accepted status is returned if the URLs are queued for deletion, or an error is returned if there are issues with the request.
func (h *DeleteHandler) DeleteBatch(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract the user ID from the request context.
	userID := req.Context().Value(middleware.UserIDKey).(string)
	if userID == "" {
		http.Error(res, "", http.StatusUnauthorized)
		return
	}

	// Parse the incoming JSON request body.
	var request []string
	err := decodeJSONBody(res, req, &request)
	if err != nil {
		// Handle malformed request errors.
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(res, mr.msg, mr.status)
			return
		}
		h.logger.Error(err.Error())
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Prepare the list of URLs to delete.
	var toDelete []storage.URLRecord
	for _, url := range request {
		toDelete = append(toDelete, storage.URLRecord{Short: url, UserID: userID})
	}

	// Perform the deletion asynchronously.
	go h.service.DeleteURLRecords(ctx, toDelete)

	// Return a 202 Accepted status to acknowledge that the deletion process is started.
	res.WriteHeader(http.StatusAccepted)
}
