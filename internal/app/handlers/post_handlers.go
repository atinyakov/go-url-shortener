package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/google/uuid"
)

type PostHandler struct {
	Resolver *services.URLResolver
	baseURL  string
	storage  storage.StorageI
	db       *sql.DB
	logger   logger.LoggerI
}

func NewPostHandler(resolver *services.URLResolver, baseURL string, s storage.StorageI, l logger.LoggerI, db *sql.DB) *PostHandler {
	return &PostHandler{
		Resolver: resolver,
		baseURL:  baseURL,
		db:       db,
		storage:  s,
		logger:   l,
	}
}

// HandlePostPlainBody handles POST requests for URL shortening
func (h *PostHandler) HandlePostPlainBody(res http.ResponseWriter, req *http.Request) {
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
	URLrecord := storage.URLRecord{Short: shortURL, Original: originalURL, ID: uuid.New().String()}

	if !exists {
		if storageErr := h.storage.Write(URLrecord); storageErr != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}

	}

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusCreated)

	_, resErr := res.Write([]byte(h.baseURL + "/" + shortURL))
	if resErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

// HandlePostJSON handles POST requests for URL shortening
func (h *PostHandler) HandlePostJSON(res http.ResponseWriter, req *http.Request) {

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

	h.logger.Info("Got Url: %s", request.URL)

	shortURL, exists := h.Resolver.LongToShort(request.URL)
	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + shortURL})

	if !exists {
		URLrecord := storage.URLRecord{Short: shortURL, Original: request.URL, ID: uuid.New().String()}

		if storageErr := h.storage.Write(URLrecord); storageErr != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}

	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

}
