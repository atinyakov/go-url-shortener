// Package models defines the request and response data structures used
// for communication between the client and the URL shortener service.
package models

// Request represents a request to shorten a URL.
type Request struct {
	// URL is the original URL to be shortened.
	URL string `json:"url"`
}

// Response represents the response containing the shortened URL.
type Response struct {
	// Result contains the shortened version of the original URL.
	Result string `json:"result"`
}

// BatchRequest represents a request to shorten multiple URLs in batch,
// each with a unique correlation ID for tracking.
type BatchRequest struct {
	// CorrelationID is a unique identifier for this request, useful for matching
	// the response with the original request.
	CorrelationID string `json:"correlation_id"`

	// OriginalURL is the URL to be shortened.
	OriginalURL string `json:"original_url"`
}

// BatchResponse represents the response for a single URL in a batch
// shortening request.
type BatchResponse struct {
	// CorrelationID matches the one sent in the corresponding BatchRequest.
	CorrelationID string `json:"correlation_id"`

	// ShortURL is the shortened version of the OriginalURL.
	ShortURL string `json:"short_url"`
}

// ByIDRequest is used for operations involving both the original and
// shortened URLs, such as deletion or lookup by ID.
type ByIDRequest struct {
	// OriginalURL is the original long-form URL.
	OriginalURL string `json:"original_url"`

	// ShortURL is the shortened version of the original URL.
	ShortURL string `json:"short_url"`
}
