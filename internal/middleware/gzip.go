package middleware

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (cw compressWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw compressWriter) Write(b []byte) (int, error) {
	return cw.zw.Write(b)
}

func (cw compressWriter) WriteHeader(statusCode int) {
	if statusCode >= 200 && statusCode < 300 {
		cw.w.Header().Del("Content-Length")
		cw.w.Header().Set("Content-Encoding", "gzip")
	} else {
		cw.w.Header().Del("Content-Encoding") // Ensure no gzip for redirects or errors
	}
	cw.w.WriteHeader(statusCode)
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func WithGZIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding := r.Header.Get("Accept-Encoding")
		contentType := r.Header.Get("Content-Type")

		// Check if the client accepts gzip and decide based on Content-Type
		if strings.Contains(acceptEncoding, "gzip") &&
			(contentType == "" || strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html")) {

			fmt.Println("inside gzip")

			// Handle gzip request body if present
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				fmt.Println("inside body gzip")

				cr, err := newCompressReader(r.Body)
				if err != nil {
					http.Error(w, "Failed to read compressed body", http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

			// Create gzip writer for compressible responses
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, "Failed to create gzip writer", http.StatusInternalServerError)
				return
			}
			defer gz.Close()

			// Wrap the ResponseWriter for gzip
			gzw := newCompressWriter(w)
			defer gzw.zw.Close()

			// Serve the handler with gzip-enabled writer
			next.ServeHTTP(gzw, r)
			return
		}

		// Pass through without compression for unsupported cases
		next.ServeHTTP(w, r)
	})
}
