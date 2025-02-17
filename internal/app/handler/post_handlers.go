package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/repository"
	"go.uber.org/zap"
)

type PostHandler struct {
	baseURL    string
	urlService *service.URLService
	logger     *zap.Logger
}

func NewPost(baseURL string, s *service.URLService, l *zap.Logger) *PostHandler {
	return &PostHandler{
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

	r, err := h.urlService.CreateURLRecord(originalURL)

	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL for %s already exists", originalURL))
			res.WriteHeader(http.StatusConflict)
			_, resErr := res.Write([]byte(h.baseURL + "/" + r.Short))
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

	r, err := h.urlService.CreateURLRecord(request.URL)

	res.Header().Set("Content-Type", "application/json")
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.logger.Info(fmt.Sprintf("URL %s already exists", request.URL))
			response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + r.Short})
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

	response, _ := json.Marshal(models.Response{Result: h.baseURL + "/" + r.Short})
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

	batchUrls, err := h.urlService.CreateURLRecords(urlsR)
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

	response, err := json.Marshal(&batchUrls)
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
