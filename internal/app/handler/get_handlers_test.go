package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func createTestHandler(mockService *mocks.MockURLServiceIface) *GetHandler {
	logger, _ := zap.NewDevelopment()
	return NewGet(mockService, logger)
}

func TestByShort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := createTestHandler(mockService)

	tests := []struct {
		name         string
		shortURL     string
		mockReturn   *storage.URLRecord
		mockErr      error
		expectedCode int
	}{
		{
			name:         "Valid URL",
			shortURL:     "abc123",
			mockReturn:   &storage.URLRecord{Original: "https://example.com", IsDeleted: false},
			mockErr:      nil,
			expectedCode: http.StatusTemporaryRedirect,
		},
		{
			name:         "URL not found",
			shortURL:     "unknown",
			mockReturn:   nil,
			mockErr:      errors.New("not found"),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Deleted URL",
			shortURL:     "deleted",
			mockReturn:   &storage.URLRecord{Original: "https://example.com", IsDeleted: true},
			mockErr:      nil,
			expectedCode: http.StatusGone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.EXPECT().GetURLByShort(gomock.Any(), tt.shortURL).Return(tt.mockReturn, tt.mockErr)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortURL, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{
					Keys:   []string{"url"},
					Values: []string{tt.shortURL},
				},
			}))
			w := httptest.NewRecorder()

			// Simulate setting URL param manually
			handler.ByShort(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

func TestPingDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := createTestHandler(mockService)

	t.Run("Success", func(t *testing.T) {
		mockService.EXPECT().PingContext(gomock.Any()).Return(nil)

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		handler.PingDB(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Failure", func(t *testing.T) {
		mockService.EXPECT().PingContext(gomock.Any()).Return(errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		handler.PingDB(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestURLsByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	handler := createTestHandler(mockService)

	t.Run("Valid user with URLs", func(t *testing.T) {
		userID := "user123"
		urls := &[]models.ByIDRequest{
			{OriginalURL: "https://example.com", ShortURL: "abc123"},
		}
		mockService.EXPECT().GetURLByUserID(gomock.Any(), userID).Return(urls, nil)

		ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
		req := httptest.NewRequest(http.MethodGet, "/user-urls", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.URLsByUserID(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("No URLs", func(t *testing.T) {
		userID := "user123"

		urls := &[]models.ByIDRequest{}
		mockService.EXPECT().GetURLByUserID(gomock.Any(), userID).Return(urls, nil)

		ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
		req := httptest.NewRequest(http.MethodGet, "/user-urls", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.URLsByUserID(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/user-urls", nil)
		w := httptest.NewRecorder()

		handler.URLsByUserID(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		userID := "user123"
		mockService.EXPECT().GetURLByUserID(gomock.Any(), userID).Return(nil, errors.New("fail"))

		ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
		req := httptest.NewRequest(http.MethodGet, "/user-urls", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.URLsByUserID(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
