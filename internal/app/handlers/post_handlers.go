package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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

	shortURL, exists, storageErr := h.Resolver.LongToShort(originalURL)

	if !exists {
		URLrecord := storage.URLRecord{Short: shortURL, Original: originalURL}

		if err := h.storage.Write(URLrecord); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}
	}

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

	fmt.Println("got URL", request.URL)
	shortURL, exists, err := h.Resolver.LongToShort(request.URL)

	if !exists {
		URLrecord := storage.URLRecord{Short: shortURL, Original: request.URL}

		if err := h.storage.Write(URLrecord); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}
	}

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

func (h *PostHandler) HandleBatch(res http.ResponseWriter, req *http.Request) {
	var urlsR []models.BatchRequest

	err := decodeJSONBody(res, req, &urlsR)
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

	var newRecords []storage.URLRecord
	var resultExisting []models.BatchResponse

	for _, url := range urlsR {
		fmt.Println("looking fof", url.CorrelationID)
		v, _ := h.storage.FindByID(url.CorrelationID)
		fmt.Println("got", v)

		if v.Short != "" {
			resultExisting = append(resultExisting, models.BatchResponse{CorrelationID: v.ID, ShortURL: h.baseURL + "/" + v.Short})
		} else {
			short, _, _ := h.Resolver.LongToShort(url.OriginalURL)

			newRecords = append(newRecords, storage.URLRecord{Original: url.OriginalURL, Short: short})

		}
	}

	var resultNew []models.BatchResponse
	fmt.Println("newRecords", newRecords)

	if len(newRecords) != 0 {

		fmt.Println("WRITE NEW REC", newRecords)

		newR, err := h.storage.WriteAll(newRecords)
		if err != nil {
			h.logger.Info(err.Error())
		}
		fmt.Println("got newr", newR)

		for _, nr := range newR {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: h.baseURL + "/" + nr.Short})
		}
	}

	fmt.Println("resultNew", resultNew)
	resultArr := append(resultExisting, resultNew...)

	response, _ := json.Marshal(resultArr)

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

}
