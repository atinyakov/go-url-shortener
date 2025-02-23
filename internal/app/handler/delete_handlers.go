package handler

import (
	"errors"
	"net/http"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

type DeleteHandler struct {
	service *service.URLService
	logger  *zap.Logger
	ch      chan<- []storage.URLRecord
}

func NewDelete(s *service.URLService, l *zap.Logger, ch chan<- []storage.URLRecord) *DeleteHandler {
	return &DeleteHandler{
		service: s,
		logger:  l,
		ch:      ch,
	}
}

func (h *DeleteHandler) DeleteBatch(res http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(middleware.UserIDKey).(string)
	if userID == "" {
		http.Error(res, "", http.StatusUnauthorized)
		return
	}

	var request []string

	err := decodeJSONBody(res, req, &request)
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(res, mr.msg, mr.status)
			return
		}
		h.logger.Error(err.Error())
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	var toDelete []storage.URLRecord
	for _, shortURL := range request {
		toDelete = append(toDelete, storage.URLRecord{Short: shortURL, UserID: userID})
	}
	h.ch <- toDelete

	res.WriteHeader(http.StatusAccepted)
}
