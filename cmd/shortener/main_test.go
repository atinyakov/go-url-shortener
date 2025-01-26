package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/app/server"
	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resolver = services.NewURLResolver(8, make(map[string]string), make(map[string]string))
var log = logger.New()

type TestStorage struct {
}

func (t TestStorage) Write(interface{}) error {
	return nil
}

func (t TestStorage) Read() ([]map[string]string, error) {
	var records []map[string]string
	return records, nil
}

var mockStorage = &TestStorage{}

func TestPostHandlers(t *testing.T) {

	req, _ := json.Marshal(models.Request{URL: "https://practicum.yandex.ru/"})

	type Request struct {
		method      string
		url         string
		body        string
		contentType string
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
				method:      http.MethodPost,
				url:         "/",
				body:        "https://practicum.yandex.ru/",
				contentType: "text/plain; charset=utf-8",
			},
			want: want{
				code:        http.StatusCreated,
				response:    `http://localhost:8080/5Ol0CyIn`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Test POST 201 JSON",
			request: Request{
				method:      http.MethodPost,
				url:         "/api/shorten",
				body:        string(req),
				contentType: "application/json",
			},
			want: want{
				code:        http.StatusCreated,
				response:    `http://localhost:8080/5Ol0CyIn`,
				contentType: "application/json",
			},
		},
		{
			name: "Test POST 400",
			request: Request{
				method:      http.MethodPost,
				url:         "/",
				body:        "",
				contentType: "text/plain; charset=utf-8",
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

	ts := httptest.NewServer(server.Init(resolver, "http://localhost:8080", log, false, mockStorage))
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new HTTP request using http.NewRequest
			// if()
			req, err := http.NewRequest(test.request.method, ts.URL+test.request.url, strings.NewReader(test.request.body))
			req.Header.Set("Content-Type", test.request.contentType)
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

			if test.want.contentType == "application/json" {
				var jsonResponse map[string]string
				err = json.Unmarshal(resBody, &jsonResponse)
				require.NoError(t, err, "error unmarshaling JSON response")
				assert.Equal(t, test.want.response, jsonResponse["result"], "unexpected JSON response field")
			} else {
				assert.Equal(t, test.want.response, string(resBody), "unexpected response body")
			}
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

	ts := httptest.NewServer(server.Init(resolver, "http://localhost:8080", log, false, mockStorage))
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
