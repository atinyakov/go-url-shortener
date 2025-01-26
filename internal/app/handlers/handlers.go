package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var mu sync.Mutex

type URLHandler struct {
	Resolver *services.URLResolver
	baseURL  string
	storage  storage.StorageI
	logger   logger.LoggerI
}

func NewURLHandler(resolver *services.URLResolver, baseURL string, s storage.StorageI, l logger.LoggerI) *URLHandler {
	return &URLHandler{
		Resolver: resolver,
		baseURL:  baseURL,
		storage:  s,
		logger:   l,
	}
}

// HandlePostPlainBody handles POST requests for URL shortening
func (h *URLHandler) HandlePostPlainBody(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	if len(originalURL) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, exists := h.Resolver.LongToShort(originalURL)
	URLrecord := models.URL{Short: shortURL, Original: originalURL, ID: uuid.New().String()}

	if !exists {
		mu.Lock()

		if storageErr := h.storage.Write(URLrecord); storageErr != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}

		mu.Unlock()
	}

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusCreated)

	_, resErr := res.Write([]byte(h.baseURL + "/" + shortURL))
	if resErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

// HandlePostJSON handles POST requests for URL shortening
func (h *URLHandler) HandlePostJSON(res http.ResponseWriter, req *http.Request) {

	var request models.Request

	err := decodeJSONBody(res, req, &request)
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(res, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	fmt.Println("got URL", request.URL)
	shortURL, _ := h.Resolver.LongToShort(request.URL)
	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + shortURL})

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
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
