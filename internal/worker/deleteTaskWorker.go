package worker

import (
	"time"

	"github.com/atinyakov/go-url-shortener/internal/app/service"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

type DeleteTaskWorker struct {
	urlSerice *service.URLService
	in        <-chan []storage.URLRecord
	logger    zap.Logger
}

func NewDeleteRecordWorker(s *service.URLService, logger zap.Logger, ch <-chan []storage.URLRecord) *DeleteTaskWorker {
	return &DeleteTaskWorker{
		urlSerice: s,
		in:        ch,
		logger:    logger,
	}
}

func (s *DeleteTaskWorker) FlushRecords() {
	// будем сохранять сообщения, накопленные за последние 100 секунд
	ticker := time.NewTicker(10 * time.Second)

	var messages []storage.URLRecord

	for {
		select {
		case msgs := <-s.in:
			s.logger.Info("Got Records to delete", zap.Any("msg", msgs))
			messages = append(messages, msgs...)
		case <-ticker.C:
			if len(messages) == 0 {
				continue
			}

			s.logger.Info("Fluching delete records", zap.Int("count=", len(messages)))
			err := s.urlSerice.DeleteURLRecords(messages)
			if err != nil {
				s.logger.Error("Cannot delete records", zap.Error(err))
				// не будем стирать сообщения, попробуем отправить их чуть позже
				continue
			}
			// сотрём успешно отосланные сообщения
			messages = nil
		}
	}
}
