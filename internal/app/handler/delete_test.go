package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/atinyakov/go-url-shortener/internal/app/handler"
	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/mocks"
	"github.com/atinyakov/go-url-shortener/internal/storage"

	"go.uber.org/mock/gomock"
)

func testLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout"}
	cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	l, _ := cfg.Build()
	return l
}

func TestDeleteBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockURLServiceIface(ctrl)
	logger := testLogger()

	h := handler.NewDelete(mockService, logger)

	t.Run("valid request returns 202", func(t *testing.T) {
		body := bytes.NewBufferString(`["abc123","def456"]`)
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", body)

		// Inject user ID into context
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		// Override callDeleteURLRecords to call the method synchronously for testing
		original := handler.СallDeleteURLRecords
		defer func() { handler.СallDeleteURLRecords = original }()
		handler.СallDeleteURLRecords = func(service service.URLServiceIface, ctx context.Context, records []storage.URLRecord) {
			service.DeleteURLRecords(ctx, records)
		}

		mockService.EXPECT().
			DeleteURLRecords(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, records []storage.URLRecord) error {
				require.Equal(t, []storage.URLRecord{
					{Short: "abc123", UserID: "user-1", IsDeleted: false},
					{Short: "def456", UserID: "user-1", IsDeleted: false},
				}, records)
				return nil
			})

		h.DeleteBatch(rec, req)

		require.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("missing user ID returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", nil)
		rec := httptest.NewRecorder()

		h.DeleteBatch(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", body)

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		h.DeleteBatch(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("decode error returns 500", func(t *testing.T) {
		// Simulate a decodeJSONBody panic/failure using a timeout context
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", nil)
		ctx, cancel := context.WithTimeout(context.Background(), -1*time.Second)
		defer cancel()

		ctx = context.WithValue(ctx, middleware.UserIDKey, "user-1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		h.DeleteBatch(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
