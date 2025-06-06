package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/middleware"
)

func TestWithSubnet(t *testing.T) {
	tests := []struct {
		name           string
		subnet         string
		realIP         string
		expectedStatus int
	}{
		{
			name:           "Allowed subnet",
			subnet:         "192.168.0",
			realIP:         "192.168.0.45",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Forbidden subnet",
			subnet:         "10.0.0",
			realIP:         "192.168.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Exact match",
			subnet:         "203.0.113.5",
			realIP:         "203.0.113.5",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing header",
			subnet:         "192.168.1",
			realIP:         "",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy handler to wrap
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the handler with the middleware
			wrapped := middleware.WithSubnet(tt.subnet)(handler)

			// Create request and set the X-Real-IP header
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
