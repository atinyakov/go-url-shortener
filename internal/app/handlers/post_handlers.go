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

	shortURL := h.Resolver.LongToShort(originalURL)

	URLrecord := storage.URLRecord{Short: shortURL, Original: originalURL}

	err = h.storage.Write(URLrecord)

	if err != nil {

		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL %s already exists", originalURL))
			res.WriteHeader(http.StatusConflict)

		} else {
			h.logger.Info(fmt.Sprintf("unable to insert row: %s", err.Error()))
			res.WriteHeader(http.StatusInternalServerError)
		}
		// return fmt.Errorf("unable to insert row: %w", err)
	} else {

		res.WriteHeader(http.StatusCreated)
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
	shortURL := h.Resolver.LongToShort(request.URL)

	// if !exists {
	URLrecord := storage.URLRecord{Short: shortURL, Original: request.URL}

	err = h.storage.Write(URLrecord)
	// if err != nil {
	// 	res.WriteHeader(http.StatusInternalServerError)
	// }

	res.Header().Set("Content-Type", "application/json")
	if err != nil {

		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL %s already exists", request.URL))
			res.WriteHeader(http.StatusConflict)

		} else {
			h.logger.Info(fmt.Sprintf("unable to insert row: %s", err.Error()))
			res.WriteHeader(http.StatusInternalServerError)
		}
		// return fmt.Errorf("unable to insert row: %w", err)
	} else {

		res.WriteHeader(http.StatusCreated)
	}
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
		} else {
			log.Print(err.Error())
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	var resultNew []models.BatchResponse

	if len(urlsR) != 0 {
		var records = make([]storage.URLRecord, 0)

		for _, url := range urlsR {
			short := h.Resolver.LongToShort(url.OriginalURL)

			records = append(records, storage.URLRecord{Original: url.OriginalURL, ID: url.CorrelationID, Short: short})
		}

		err := h.storage.WriteAll(records)
		if err != nil {
			h.logger.Info(err.Error())
		}

		// if err != nil {
		// 	var pgErr *pgconn.PgError
		// 	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		// 		fmt.Println("UniqueViolation")
		// 		h.logger.Info(err.Error())

		// 	}
		// }

		for _, nr := range records {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: h.baseURL + "/" + nr.Short})
		}
	}

	response, _ := json.Marshal(resultNew)

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	_, writeErr := res.Write(response)
	if writeErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}
