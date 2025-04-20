package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func BenchmarkPostPlainBody(b *testing.B) {
	var mockStorage, _ = storage.CreateMemoryStorage()

	var resolver, _ = service.NewURLResolver(8, mockStorage)
	log := logger.New()
	zapLogger := log.Log

	var URLService = service.NewURL(mockStorage, resolver, zapLogger, "http://localhost:8080")
	// Инициализация обработчика
	postHandler := NewPost("http://localhost", URLService, zapLogger)

	// Создаём запрос
	body := []byte("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()

	b.ResetTimer()

	// Бенчмаркаем несколько запросов
	for i := 0; i < b.N; i++ {
		postHandler.PlainBody(w, req)
	}
}

func BenchmarkPostJSON(b *testing.B) {

	var mockStorage, _ = storage.CreateMemoryStorage()

	var resolver, _ = service.NewURLResolver(8, mockStorage)
	log := logger.New()
	zapLogger := log.Log

	var URLService = service.NewURL(mockStorage, resolver, zapLogger, "http://localhost:8080")

	// Инициализация обработчика
	postHandler := NewPost("http://localhost", URLService, zapLogger)

	// Создаём запрос с JSON телом
	reqBody := models.Request{URL: "https://example.com"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	b.ResetTimer()

	// Бенчмаркаем несколько запросов
	for i := 0; i < b.N; i++ {
		postHandler.HandlePostJSON(w, req)
	}
}
