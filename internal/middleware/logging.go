package middleware

import (
	"net/http"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/logger"
)

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

func WithLogging(log logger.LoggerI) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   responseData,
			}

			next.ServeHTTP(&lw, r) // внедряем реализацию http.ResponseWriter

			duration := time.Since(start)

			log.Info("HTTP Request",
				"method", r.Method,
				"url", r.URL.String(),
				"encoding", r.Header.Get("Accept-Encoding"),
				"contentType", r.Header.Get("Content-Type"),
				"duration", time.Since(start),
				"status", responseData.status, // получаем перехваченный код статуса ответа
				"duration", duration,
				"size", responseData.size, // получаем перехваченный размер ответа
			)

		})
	}
}
