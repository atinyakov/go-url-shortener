// Package handler provides HTTP handlers for URL shortening services.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/repository"
)

// PostHandler handles POST requests for URL shortening.
type PostHandler struct {
	baseURL    string                  // Base URL for the service.
	urlService service.URLServiceIface // Service for handling URL operations.
	logger     *zap.Logger             // Logger for logging events.
}

// NewPost creates a new PostHandler instance with provided base URL, URL service, and logger.
func NewPost(baseURL string, s service.URLServiceIface, l *zap.Logger) *PostHandler {
	return &PostHandler{
		baseURL:    baseURL,
		urlService: s,
		logger:     l,
	}
}

// PlainBody handles POST requests for URL shortening when the body contains a plain URL string.
// The URL will be shortened and returned in the response body.
func (h *PostHandler) PlainBody(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract user ID from request context.
	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "User ID not found", http.StatusInternalServerError)
		return
	}

	// Read the request body.
	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil || len(body) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Call the service to create a new shortened URL.
	originalURL := string(body)
	r, err := h.urlService.CreateURLRecord(ctx, originalURL, userID)

	// Handle different errors and responses.
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info("URL already exists", zap.String("originalURL", originalURL))
			res.WriteHeader(http.StatusConflict)
			_, resErr := res.Write([]byte(h.baseURL + "/" + r.Short))
			if resErr != nil {
				res.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		h.logger.Info("unable to insert row:", zap.String("error", err.Error()))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Return the shortened URL in the response.
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusCreated)
	_, resErr := res.Write([]byte(h.baseURL + "/" + r.Short))
	if resErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

// HandlePostJSON handles POST requests for URL shortening when the body contains JSON data.
// The request expects a JSON body with a URL to shorten, and the response will contain the shortened URL in JSON format.
func (h *PostHandler) HandlePostJSON(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract user ID from request context.
	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "User ID not found", http.StatusInternalServerError)
		return
	}

	// Decode the request body into the Request model.
	var request models.Request
	err := decodeJSONBody(res, req, &request)
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(res, mr.msg, mr.status)
			return
		}
		log.Print(err.Error())
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Create a new shortened URL using the URL service.
	r, err := h.urlService.CreateURLRecord(ctx, request.URL, userID)

	// Handle errors and send appropriate responses.
	res.Header().Set("Content-Type", "application/json")
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info("URL already exists", zap.String("originalURL", request.URL))
			response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + r.Short})
			res.WriteHeader(http.StatusConflict)
			_, writeErr := res.Write(response)
			if writeErr != nil {
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			return
		}
		h.logger.Info("unable to insert row:", zap.String("error", err.Error()))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Return the shortened URL in JSON format.
	res.WriteHeader(http.StatusCreated)
	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + r.Short})
	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

// HandleBatch handles POST requests for batch URL shortening.
// The request expects a JSON body with a list of URLs to shorten, and the response will contain a JSON array with shortened URLs.
func (h *PostHandler) HandleBatch(res http.ResponseWriter, req *http.Request) {
	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	// Extract user ID from request context.
	userID, ok := req.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(res, "User ID not found", http.StatusInternalServerError)
		return
	}

	// Decode the request body into the BatchRequest model.
	var urlsR []models.BatchRequest
	err := decodeJSONBody(res, req, &urlsR)
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(res, mr.msg, mr.status)
			return
		}
		log.Print(err.Error())
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Call the service to create a batch of shortened URLs.
	batchUrls, err := h.urlService.CreateURLRecords(ctx, urlsR, userID)
	if errors.Is(err, repository.ErrConflict) {
		h.logger.Info(err.Error())
		res.WriteHeader(http.StatusConflict)
		return
	}

	if err != nil {
		h.logger.Info(err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Return the list of shortened URLs in JSON format.
	response, err := json.Marshal(&batchUrls)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}
