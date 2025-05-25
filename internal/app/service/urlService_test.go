package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func TestURLService_CreateURLRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var mockStorage, _ = storage.CreateMemoryStorage()

	var mockResolver, _ = NewURLResolver(8, mockStorage)
	mockLogger := zap.NewNop()

	service, _ := NewURL(context.Background(), mockStorage, mockResolver, mockLogger, "http://baseurl")

	result, err := service.CreateURLRecord(context.Background(), "http://example.com", "user-id")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "http://example.com", result.Original)
	assert.Equal(t, "h1ZwLLGa", result.Short)
}

func TestURLService_CreateURLRecords(t *testing.T) {
	mockStorage, _ := storage.CreateMemoryStorage()
	mockResolver, _ := NewURLResolver(8, mockStorage)
	mockLogger := zap.NewNop()

	ctx := context.Background()

	// Input
	batchRequest := []models.BatchRequest{
		{OriginalURL: "http://example.com", CorrelationID: "123"},
	}
	userID := "user-id"

	// Service
	service, _ := NewURL(context.Background(), mockStorage, mockResolver, mockLogger, "http://baseurl")

	// Act
	result, _ := service.CreateURLRecords(ctx, batchRequest, userID)

	// Assert
	responses := *result
	assert.Len(t, responses, 1)
	assert.Contains(t, responses[0].ShortURL, "http://baseurl/")
	assert.Equal(t, "123", responses[0].CorrelationID)

	// Also check if it was stored
	stored, err := mockStorage.FindByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, *stored, 1)
	assert.Equal(t, "http://example.com", (*stored)[0].Original)
}

func TestURLService_GetURLByShort(t *testing.T) {
	mockStorage, _ := storage.CreateMemoryStorage()
	mockResolver, _ := NewURLResolver(8, mockStorage)
	mockLogger := zap.NewNop()

	_ = mockStorage.WriteAll(context.Background(), []storage.URLRecord{
		{
			Original: "http://example.com",
			Short:    "short-url",
			UserID:   "user-id",
		},
	})

	service, _ := NewURL(context.Background(), mockStorage, mockResolver, mockLogger, "http://baseurl")

	result, err := service.GetURLByShort(context.Background(), "short-url")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "http://example.com", result.Original)
	assert.Equal(t, "short-url", result.Short)
}

func TestURLService_GetURLByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage, _ := storage.CreateMemoryStorage()
	mockResolver, _ := NewURLResolver(8, mockStorage)
	mockLogger := zap.NewNop()

	_, err := mockStorage.Write(context.Background(), storage.URLRecord{
		Original: "http://example.com",
		Short:    "short-url",
		UserID:   "user-id",
	})
	require.NoError(t, err)

	service, _ := NewURL(context.Background(), mockStorage, mockResolver, mockLogger, "http://baseurl")

	result, err := service.GetURLByUserID(context.Background(), "user-id")

	// Assertions
	assert.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, *result, 1)
	assert.Equal(t, "http://baseurl/short-url", (*result)[0].ShortURL)
	assert.Equal(t, "http://example.com", (*result)[0].OriginalURL)
}

func TestURLService_PingContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage, _ := storage.CreateMemoryStorage()
	mockResolver, _ := NewURLResolver(8, mockStorage)
	mockLogger := zap.NewNop()

	service, _ := NewURL(context.Background(), mockStorage, mockResolver, mockLogger, "http://baseurl")

	err := service.PingContext(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
