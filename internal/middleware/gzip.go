package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type GzipResponseWriter struct {
	Writer io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// func (w GzipResponseWriter) WriteHeader(statusCode int) {
// 	if statusCode >= 200 && statusCode < 300 {
// 		w.Header().Del("Content-Length")
// 		w.Header().Set("Content-Encoding", "gzip")
// 	} else {
// 		w.Header().Del("Content-Encoding") // Ensure no gzip for redirects or errors
// 	}
// 	w.WriteHeader(statusCode)
// }

// type compressReader struct {
// 	r  io.ReadCloser
// 	zr *gzip.Reader
// }

// func newCompressReader(r io.ReadCloser) (*compressReader, error) {
// 	zr, err := gzip.NewReader(r)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &compressReader{
// 		r:  r,
// 		zr: zr,
// 	}, nil
// }

// func (c compressReader) Read(p []byte) (n int, err error) {
// 	return c.zr.Read(p)
// }

// func (c *compressReader) Close() error {
// 	if err := c.r.Close(); err != nil {
// 		return err
// 	}
// 	return c.zr.Close()
// }

func WithGZIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptsEncoding := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		contentType := r.Header.Get("Content-Type")
		isPlainText := strings.Contains(contentType, "text/plain")
		// sendsEncoded := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")

		// Check if the client accepts gzip and decide based on Content-Type
		// Handle gzip request body if present
		// if sendsEncoded {
		// 	reader, err := gzip.NewReader(r.Body)
		// 	if err != nil {
		// 		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		// 		return
		// 	}
		// 	defer reader.Close()
		// 	r.Body = io.NopCloser(reader)
		// }

		if acceptsEncoding && !isPlainText {
			w.Header().Set("Content-Encoding", "gzip")

			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, "Failed to create gzip writer", http.StatusInternalServerError)
				return
			}
			defer gz.Close()

			gzw := GzipResponseWriter{Writer: gz, ResponseWriter: w}
			// defer gzw.zw.Close()

			next.ServeHTTP(gzw, r)
			return
		}

		// Pass through without compression for unsupported cases
		next.ServeHTTP(w, r)
	})
}
