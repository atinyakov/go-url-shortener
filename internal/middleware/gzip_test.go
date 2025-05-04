package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithGZIPGet(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		contentType    string
		expectGzip     bool
	}{
		{"gzip accepted, not text/plain", "gzip", "application/json", true},
		{"gzip accepted, text/plain", "gzip", "text/plain", false},
		{"no gzip accepted", "", "application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("hello world"))
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			req.Header.Set("Content-Type", tt.contentType)

			rec := httptest.NewRecorder()
			WithGZIPGet(handler).ServeHTTP(rec, req)
			resp := rec.Result()
			defer resp.Body.Close()

			// Check if GZIP header is set
			encoding := resp.Header.Get("Content-Encoding")
			if tt.expectGzip {
				if encoding != "gzip" {
					t.Errorf("expected gzip encoding, got %s", encoding)
				}

				// Check if response is valid GZIP data
				gr, err := gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatalf("failed to read gzip body: %v", err)
				}
				defer gr.Close()
				unzipped, err := io.ReadAll(gr)
				if err != nil {
					t.Fatalf("failed to decompress body: %v", err)
				}
				if string(unzipped) != "hello world" {
					t.Errorf("unexpected body: %s", unzipped)
				}
			} else {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != "hello world" {
					t.Errorf("unexpected body: %s", body)
				}
				if encoding != "" {
					t.Errorf("expected no Content-Encoding, got %s", encoding)
				}
			}
		})
	}
}

func TestWithGZIPPost(t *testing.T) {
	t.Run("valid gzip request", func(t *testing.T) {
		var bodyBuf bytes.Buffer
		gzw := gzip.NewWriter(&bodyBuf)
		_, _ = gzw.Write([]byte("decompressed content"))
		gzw.Close()

		req := httptest.NewRequest(http.MethodPost, "/", &bodyBuf)
		req.Header.Set("Content-Encoding", "gzip")

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read body: %v", err)
			}
			if string(b) != "decompressed content" {
				t.Errorf("unexpected decompressed body: %s", b)
			}
			w.WriteHeader(http.StatusOK)
		})

		WithGZIPPost(handler).ServeHTTP(rec, req)
		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}

	})

	t.Run("invalid gzip request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzip data"))
		req.Header.Set("Content-Encoding", "gzip")

		rec := httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called on invalid gzip")
		})

		WithGZIPPost(handler).ServeHTTP(rec, req)
		resp := rec.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 BadRequest, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Failed to decompress") {
			t.Errorf("unexpected error body: %s", body)
		}
	})
}
