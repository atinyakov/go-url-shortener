package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resolver = services.NewURLResolver(8)
var log = logger.New()

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
				url:    "/",
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
				url:    "/",
				body:   "",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "",
			},
		},
	}
	err := log.Init("Info")
	require.NoError(t, err)

	ts := httptest.NewServer(server.Init(resolver, "http://localhost:8080", log))
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new HTTP request using http.NewRequest
			req, err := http.NewRequest(test.request.method, ts.URL+test.request.url, strings.NewReader(test.request.body))
			require.NoError(t, err)

			// Send the request using the test server's client
			result, err := ts.Client().Do(req)
			require.NoError(t, err)
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
				url:    "/",
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
				url:    "/123",
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
				url:    "/5Ol0CyIn",
			},
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "https://practicum.yandex.ru/",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	err := log.Init("Info")
	require.NoError(t, err)

	ts := httptest.NewServer(server.Init(resolver, "http://localhost:8080", log))
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new HTTP request using http.NewRequest
			t.Logf("Requesting URL: %s", ts.URL+test.request.url)
			req, err := http.NewRequest(test.request.method, ts.URL+test.request.url, nil)
			require.NoError(t, err)

			// Send the request using the test server's client
			client := ts.Client()
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}

			result, err := client.Do(req)
			require.NoError(t, err)
			defer result.Body.Close()

			// Assert response status code
			assert.Equal(t, test.want.code, result.StatusCode, "unexpected status code")

			// Assert response location header (for redirects)
			assert.Equal(t, test.want.location, result.Header.Get("Location"), "unexpected location header")
		})
	}
}
