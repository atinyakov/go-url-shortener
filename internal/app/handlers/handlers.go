package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	Resolver *services.URLResolver
}

func NewURLHandler(resolver *services.URLResolver) *URLHandler {
	return &URLHandler{
		Resolver: resolver,
	}
}

// HandlePost handles POST requests for URL shortening
func (h *URLHandler) HandlePost(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(string(body)) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL := h.Resolver.LongToShort(string(body))

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write([]byte("http://localhost:8080/" + shortURL))
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

// HandleGet handles GET requests for URL resolution
func (h *URLHandler) HandleGet(res http.ResponseWriter, req *http.Request) {
	shortURL := chi.URLParam(req, "url")

	fmt.Println("sorturl", shortURL)

	longURL := h.Resolver.ShortToLong(shortURL)
	if longURL == "" {
		http.Error(res, "URL not found", http.StatusNotFound)
		return
	}

	res.Header().Set("Location", longURL)

	res.WriteHeader(http.StatusTemporaryRedirect)
}
