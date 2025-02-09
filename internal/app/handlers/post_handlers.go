package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/services"
	"github.com/atinyakov/go-url-shortener/internal/logger"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type PostHandler struct {
	Resolver *services.URLResolver
	baseURL  string
	storage  storage.StorageI
	logger   logger.LoggerI
}

func NewPostHandler(resolver *services.URLResolver, baseURL string, s storage.StorageI, l logger.LoggerI) *PostHandler {
	return &PostHandler{
		Resolver: resolver,
		baseURL:  baseURL,
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

	shortURL, storageErr := h.Resolver.LongToShort(originalURL)

	if storageErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
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

	shortURL, err := h.Resolver.LongToShort(request.URL)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + shortURL})

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

}
