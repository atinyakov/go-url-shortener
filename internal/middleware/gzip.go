package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

type GzipResponseWriter struct {
	Writer io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func WithGZIPGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptsEncoding := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		isPlainText := strings.Contains(r.Header.Get("Content-Type"), "text/plain")

		if acceptsEncoding && !isPlainText {
			w.Header().Set("Content-Encoding", "gzip")

			gz := gzipWriterPool.Get().(*gzip.Writer)
			gz.Reset(w)

			defer func() {
				gz.Close()
				gzipWriterPool.Put(gz)
			}()

			gzw := GzipResponseWriter{Writer: gz, ResponseWriter: w}

			next.ServeHTTP(gzw, r)
			return
		}

		// Pass through without compression for unsupported cases
		next.ServeHTTP(w, r)
	})
}
func WithGZIPPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendsEncoded := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

		// Check if the client accepts gzip and decide based on Content-Type
		// Handle gzip request body if present
		if sendsEncoded {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = io.NopCloser(reader)
		}

		// Pass through without compression for unsupported cases
		next.ServeHTTP(w, r)
	})
}
