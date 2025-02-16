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
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type PostHandler struct {
	Resolver   *services.URLResolver
	baseURL    string
	urlService *services.URLService
	logger     *logger.Logger
}

func NewPostHandler(resolver *services.URLResolver, baseURL string, s *services.URLService, l *logger.Logger) *PostHandler {
	return &PostHandler{
		Resolver:   resolver,
		baseURL:    baseURL,
		urlService: s,
		logger:     l,
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

	shortURL := h.Resolver.LongToShort(originalURL)

	r, err := h.urlService.CreateURLRecord(storage.URLRecord{Short: shortURL, Original: originalURL})

	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL for %s already exists", originalURL))
			res.WriteHeader(http.StatusConflict)
			_, resErr := res.Write([]byte(h.baseURL + "/" + shortURL))
			if resErr != nil {
				res.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		h.logger.Info(fmt.Sprintf("unable to insert row: %s", err.Error()))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusCreated)

	_, resErr := res.Write([]byte(h.baseURL + "/" + r.Short))
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

	shortURL := h.Resolver.LongToShort(request.URL)

	_, err = h.urlService.CreateURLRecord(storage.URLRecord{Short: shortURL, Original: request.URL})

	res.Header().Set("Content-Type", "application/json")
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL %s already exists", request.URL))
			response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + shortURL})
			res.WriteHeader(http.StatusConflict)
			_, writeErr := res.Write(response)
			if writeErr != nil {
				res.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		h.logger.Info(fmt.Sprintf("unable to insert row: %s", err.Error()))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusCreated)

	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + shortURL})
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
			return
		}

		log.Print(err.Error())
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var resultNew []models.BatchResponse

	if len(urlsR) != 0 {
		var records = make([]storage.URLRecord, 0)

		for _, url := range urlsR {
			short := h.Resolver.LongToShort(url.OriginalURL)

			records = append(records, storage.URLRecord{Original: url.OriginalURL, ID: url.CorrelationID, Short: short})
		}

		err := h.urlService.CreateURLRecords(records)
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(err.Error())
			res.WriteHeader(http.StatusConflict)
			return
		}

		if err != nil {
			h.logger.Info(err.Error())
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, nr := range records {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: h.baseURL + "/" + nr.Short})
		}
	}

	response, err := json.Marshal(resultNew)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}
