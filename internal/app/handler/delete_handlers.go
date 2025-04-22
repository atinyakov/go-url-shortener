package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/middleware"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type DeleteHandler struct {
	service service.URLServiceIface
	logger  *zap.Logger
}

func NewDelete(s service.URLServiceIface, l *zap.Logger) *DeleteHandler {
	return &DeleteHandler{
		service: s,
		logger:  l,
	}
}

func (h *DeleteHandler) DeleteBatch(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

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
	for _, url := range request {
		toDelete = append(toDelete, storage.URLRecord{Short: url, UserID: userID})
	}

	go h.service.DeleteURLRecords(ctx, toDelete)

	res.WriteHeader(http.StatusAccepted)
}
