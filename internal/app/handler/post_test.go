package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func newTestPostHandler(t *testing.T) *PostHandler {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// Mock the logger
	logger, _ := zap.NewProduction()

	// Mock the service
	mockService := mocks.NewMockURLServiceIface(ctrl)

	// Create PostHandler
	return &PostHandler{
		urlService: mockService,
		logger:     logger,
		baseURL:    "http://localhost:8080",
	}
}
func TestPlainBody(t *testing.T) {
	handler := newTestPostHandler(t)

	tests := []struct {
		name                    string
		body                    string
		mockResponse            *storage.URLRecord
		mockCreateError         error
		mockGetExistingResponse *storage.URLRecord
		mockGetExistingError    error
		expectedCode            int
		expectedBody            string
	}{
		{
			name:                    "Valid URL",
			body:                    "https://example.com",
			mockResponse:            &storage.URLRecord{Short: "abc123"},
			mockCreateError:         nil,
			mockGetExistingResponse: nil, // No conflict here
			mockGetExistingError:    nil,
			expectedCode:            http.StatusCreated,
			expectedBody:            "http://localhost:8080/abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the CreateURLRecord method
			handler.urlService.(*mocks.MockURLServiceIface).EXPECT().
				CreateURLRecord(gomock.Any(), tt.body, "test-user-id").
				Return(tt.mockResponse, tt.mockCreateError).
				Times(1)

			req, err := http.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(tt.body))
			req = middleware.InjectUserID(req, "test-user-id")

			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.PlainBody(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestHandlePostJSON(t *testing.T) {
	handler := newTestPostHandler(t)

	tests := []struct {
		name         string
		body         string
		mockResponse *storage.URLRecord
		mockError    error
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid JSON",
			body:         `{"url":"https://example.com"}`,
			mockResponse: &storage.URLRecord{Short: "abc123"},
			mockError:    nil,
			expectedCode: http.StatusCreated,
			expectedBody: `{"result":"http://localhost:8080/abc123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the CreateURLRecord method
			handler.urlService.(*mocks.MockURLServiceIface).EXPECT().
				CreateURLRecord(gomock.Any(), "https://example.com", "test-user-id").
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			req, err := http.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(tt.body))
			req = middleware.InjectUserID(req, "test-user-id")

			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandlePostJSON(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}
