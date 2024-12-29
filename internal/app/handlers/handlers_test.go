package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resolver = services.NewURLResolver(8)

func TestPostHandlers(t *testing.T) {
	type Request struct {
		method string
		url    string
		body   string
	}

	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name    string
		request Request
		want    want
	}{
		{
			name: "Test POST 201",
			request: Request{
				method: http.MethodPost,
				url:    "http://localhost:8080/",
				body:   "https://practicum.yandex.ru/",
			},
			want: want{
				code:        http.StatusCreated,
				response:    `http://localhost:8080/5Ol0CyIn`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Test POST 400",
			request: Request{
				method: http.MethodPost,
				url:    "http://localhost:8080/",
				body:   "",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new HTTP request and response recorder
			r := httptest.NewRequest(test.request.method, test.request.url, strings.NewReader(test.request.body))
			w := httptest.NewRecorder()

			handler := NewURLHandler(resolver)

			// Call the handler
			handler.HandlePost(w, r)

			// Get the result
			result := w.Result()

			// Ensure the response body is closed
			defer result.Body.Close()

			// Assert response status code
			assert.Equal(t, test.want.code, result.StatusCode, "unexpected status code")

			// Assert response content type
			assert.Equal(t, test.want.contentType, result.Header.Get("Content-Type"), "unexpected content type")

			// Assert response body
			resBody, err := io.ReadAll(result.Body)
			require.NoError(t, err, "error reading response body")
			assert.Equal(t, test.want.response, string(resBody), "unexpected response body")
		})
	}
}

func TestGetHandlers(t *testing.T) {
	type Request struct {
		method string
		url    string
		body   string
	}

	type want struct {
		code        int
		location    string
		contentType string
	}
	tests := []struct {
		name    string
		request Request
		want    want
	}{
		{
			name: "Test GET 400",
			request: Request{
				method: http.MethodGet,
				url:    "http://localhost:8080/",
				body:   "",
			},
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Test GET 404",
			request: Request{
				method: http.MethodGet,
				url:    "http://localhost:8080/123",
				body:   "",
			},
			want: want{
				code:        http.StatusNotFound,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Test GET 307",
			request: Request{
				method: http.MethodGet,
				url:    "http://localhost:8080/5Ol0CyIn",
				body:   "",
			},
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "https://practicum.yandex.ru/",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new HTTP request and response recorder
			r := httptest.NewRequest(test.request.method, test.request.url, strings.NewReader(test.request.body))
			w := httptest.NewRecorder()

			handler := NewURLHandler(resolver)

			// Call the handler
			handler.HandleGet(w, r)

			// Get the result
			result := w.Result()

			// Ensure response body is always closed
			defer func() {
				if result.Body != nil {
					result.Body.Close()
				}
			}()

			// Assert response status code
			assert.Equal(t, test.want.code, result.StatusCode, "unexpected status code")

			// Assert response content type
			assert.Equal(t, test.want.contentType, result.Header.Get("Content-Type"), "unexpected content type")

			// Assert response location header (for redirects)
			assert.Equal(t, test.want.location, result.Header.Get("Location"), "unexpected location header")
		})
	}
}
