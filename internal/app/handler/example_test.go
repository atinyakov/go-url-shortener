package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/atinyakov/go-url-shortener/internal/app/handler"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func setupMockService(t *testing.T) (*mocks.MockURLServiceIface, service.URLServiceIface) {
	ctrl := gomock.NewController(t)
	mockStorage, _ := storage.CreateMemoryStorage()
	resolver, _ := service.NewURLResolver(8, mockStorage)
	zapLogger := logger.New().Log
	urlService := service.NewURL(mockStorage, resolver, zapLogger, "http://localhost:8080")
	mockService := mocks.NewMockURLServiceIface(ctrl)
	return mockService, urlService
}

func TestPlainBody(t *testing.T) {
	mockService, _ := setupMockService(t)
	h := handler.NewPost("http://localhost:8080", mockService, logger.New().Log)

	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-123"))
	rec := httptest.NewRecorder()

	expected := &storage.URLRecord{Original: reqBody, Short: "abc123", UserID: "user-123"}
	mockService.EXPECT().CreateURLRecord(gomock.Any(), reqBody, "user-123").Return(expected, nil)

	h.PlainBody(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), expected.Short)
}

func TestHandlePostJSON(t *testing.T) {
	mockService, _ := setupMockService(t)
	h := handler.NewPost("http://localhost:8080", mockService, logger.New().Log)

	payload := models.Request{URL: "https://example.com"}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-123"))
	rec := httptest.NewRecorder()

	expected := &storage.URLRecord{Original: payload.URL, Short: "abc123", UserID: "user-123"}
	mockService.EXPECT().CreateURLRecord(gomock.Any(), payload.URL, "user-123").Return(expected, nil)

	h.HandlePostJSON(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var resp models.Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.Contains(t, resp.Result, expected.Short)
}

func setupMockGetService(t *testing.T) (*mocks.MockURLServiceIface, service.URLServiceIface) {
	ctrl := gomock.NewController(t)

	mockStorage, _ := storage.CreateMemoryStorage()
	resolver, _ := service.NewURLResolver(8, mockStorage)
	log := zap.NewNop()

	urlService := service.NewURL(mockStorage, resolver, log, "http://localhost:8080")
	mockService := mocks.NewMockURLServiceIface(ctrl)

	return mockService, urlService
}

func TestByShort(t *testing.T) {
	mockService, _ := setupMockGetService(t)
	h := handler.NewGet(mockService, zap.NewNop())

	r := &storage.URLRecord{Original: "http://example.com", Short: "abc123", IsDeleted: false}
	mockService.EXPECT().GetURLByShort(gomock.Any(), "abc123").Return(r, nil)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req = muxRequestWithParam(req, "url", "abc123")

	w := httptest.NewRecorder()
	h.ByShort(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	require.Equal(t, r.Original, resp.Header.Get("Location"))
}

func TestPingDB(t *testing.T) {
	mockService, _ := setupMockGetService(t)
	h := handler.NewGet(mockService, zap.NewNop())

	mockService.EXPECT().PingContext(gomock.Any()).Return(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	h.PingDB(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestURLsByUserID(t *testing.T) {
	mockService, _ := setupMockGetService(t)
	h := handler.NewGet(mockService, zap.NewNop())

	urls := []storage.URLRecord{{Short: "short", Original: "original"}}
	mockService.EXPECT().GetURLByUserID(gomock.Any(), "test-user").Return(&urls, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "test-user"))
	w := httptest.NewRecorder()

	h.URLsByUserID(w, req)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result []storage.URLRecord
	require.NoError(t, json.Unmarshal(body, &result))
	require.Len(t, result, 1)
	require.Equal(t, "short", result[0].Short)
}

// muxRequestWithParam simulates chi's URLParam extraction
func muxRequestWithParam(r *http.Request, key, value string) *http.Request {
	tctx := chi.NewRouteContext()
	tctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, tctx))
}
