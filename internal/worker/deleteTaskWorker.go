package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type Repo interface {
	DeleteBatch(context.Context, []storage.URLRecord) error
}

type DeleteTaskWorker struct {
	in     chan storage.URLRecord
	logger *zap.Logger
	repo   Repo
}

func NewDeleteRecordWorker(logger *zap.Logger, repo Repo) *DeleteTaskWorker {
	ch := make(chan storage.URLRecord)

	return &DeleteTaskWorker{
		in:     ch,
		logger: logger,
		repo:   repo,
	}
}

func (s *DeleteTaskWorker) GetInChannel() chan<- storage.URLRecord {
	s.logger.Info("get in channle")

	return s.in
}

func (s *DeleteTaskWorker) FlushRecords() {
	// будем сохранять сообщения, накопленные за последние 100 секунд
	s.logger.Info("Fluching records init")
	ticker := time.NewTicker(10 * time.Second)
	var messages []storage.URLRecord

	sendMessages := func() {
		s.logger.Info("Fluching delete records", zap.Int("count=", len(messages)))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := s.repo.DeleteBatch(ctx, messages)

		if err != nil {
			s.logger.Error("Cannot delete records", zap.Error(err))
			messages = messages[:0]

			return
		}
		// сотрём успешно отосланные сообщения
		messages = messages[:0]
	}

	for {
		select {
		case msg := <-s.in:
			s.logger.Info("Got Records to delete", zap.Any("msg", msg))
			messages = append(messages, msg)
			if len(messages) > 25 {
				sendMessages()
			}
		case <-ticker.C:
			if len(messages) == 0 {
				continue
			}
			sendMessages()
		}
	}
}
