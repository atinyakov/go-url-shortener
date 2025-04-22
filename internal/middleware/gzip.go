// Package middleware provides HTTP middleware for handling GZIP compression
// and decompression for both incoming and outgoing HTTP requests.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	// New function creates a new gzip.Writer, which will be pooled for reuse
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

// GzipResponseWriter is a custom http.ResponseWriter that writes to a GZIP writer.
// It is used to compress the response body before sending it to the client.
type GzipResponseWriter struct {
	Writer io.Writer
	http.ResponseWriter
}

// Write writes the compressed data to the GZIP writer.
func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// WithGZIPGet is an HTTP middleware that compresses the response body using GZIP
// when the client supports GZIP compression and the content type is not plain text.
// It is intended for GET requests.
func WithGZIPGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the client supports GZIP compression and the content is not plain text
		acceptsEncoding := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		isPlainText := strings.Contains(r.Header.Get("Content-Type"), "text/plain")

		// If GZIP is supported and content type is not plain text, compress the response
		if acceptsEncoding && !isPlainText {
			w.Header().Set("Content-Encoding", "gzip")

			// Get a GZIP writer from the pool
			gz := gzipWriterPool.Get().(*gzip.Writer)
			gz.Reset(w)

			// Ensure the GZIP writer is closed after use
			defer func() {
				gz.Close()
				gzipWriterPool.Put(gz) // Return the writer to the pool
			}()

			// Wrap the original ResponseWriter with the GZIP writer
			gzw := GzipResponseWriter{Writer: gz, ResponseWriter: w}

			// Pass the GZIP-wrapped writer to the next handler
			next.ServeHTTP(gzw, r)
			return
		}

		// Pass through without compression for unsupported cases
		next.ServeHTTP(w, r)
	})
}

// WithGZIPPost is an HTTP middleware that decompresses the request body if the client
// sends a GZIP-encoded request. It is intended for POST requests.
func WithGZIPPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request body is GZIP-encoded
		sendsEncoded := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

		// If GZIP is detected in the request body, decompress it
		if sendsEncoded {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = io.NopCloser(reader) // Replace the request body with the decompressed reader
		}

		// Pass through without decompression for unsupported cases
		next.ServeHTTP(w, r)
	})
}
