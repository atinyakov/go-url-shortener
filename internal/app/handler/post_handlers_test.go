package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
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
func TestHandleBatch(t *testing.T) {
	handler := newTestPostHandler(t)

	tests := []struct {
		name         string
		body         string
		mockResponse []models.BatchResponse
		mockError    error
		expectedCode int
		expectedBody string
	}{
		{
			name: "Valid Batch Request",
			body: `[{
						"correlation_id": "6ab2e9c1-d8bf-4522-cf80-48a4d9844126",
						"original_url": "https://github.com/atinyakov/go-url-shortener/actions/runs/13423527776/job/37501582498?pr=15"
					},
					{
						"correlation_id": "9eb7d9bb-2a42-4060-9f2d-884302735503",
						"original_url": "https://github.com/atinyakov/go-url-shortener"
					}
				]`,
			mockResponse: []models.BatchResponse{
				{CorrelationID: "6ab2e9c1-d8bf-4522-cf80-48a4d9844126", ShortURL: "http://localhost:8080/abc123"},
				{CorrelationID: "9eb7d9bb-2a42-4060-9f2d-884302735503", ShortURL: "http://localhost:8080/def456"},
			},
			mockError:    nil,
			expectedCode: http.StatusCreated,
			expectedBody: `[{"correlation_id":"6ab2e9c1-d8bf-4522-cf80-48a4d9844126","short_url":"http://localhost:8080/abc123"},
							 {"correlation_id":"9eb7d9bb-2a42-4060-9f2d-884302735503","short_url":"http://localhost:8080/def456"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock.MatchedBy to match context correctly
			handler.urlService.(*mocks.MockURLServiceIface).
				EXPECT().
				CreateURLRecords(mock.MatchedBy(func(ctx context.Context) bool {
					// Ensure the context contains the expected user ID
					if val := ctx.Value(middleware.UserIDKey); val == "test-user-id" {
						return true
					}
					return false
				}), gomock.Any(), "test-user-id"). // Ensure the batch request is passed as the second argument
				Return(&tt.mockResponse, tt.mockError).
				Times(1)

			req, err := http.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(tt.body))
			req = middleware.InjectUserID(req, "test-user-id")

			if err != nil {
				t.Fatal(err)
			}

			// Set headers (if any)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleBatch(rr, req)

			// Ensure the correct status code
			assert.Equal(t, tt.expectedCode, rr.Code)

			// Normalize expected and actual response bodies by stripping whitespace and comparing
			assert.JSONEq(t, strings.ReplaceAll(tt.expectedBody, "\n", ""), strings.ReplaceAll(rr.Body.String(), "\n", ""))
		})
	}
}
