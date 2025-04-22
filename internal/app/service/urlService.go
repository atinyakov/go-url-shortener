package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/atinyakov/go-url-shortener/internal/worker"
)

type URLService struct {
	repository Storage
	resolver   *URLResolver
	logger     *zap.Logger
	baseURL    string
	ch         chan<- storage.URLRecord
}

func NewURL(repo Storage, resolver *URLResolver, logger *zap.Logger, baseURL string) *URLService {
	worker := worker.NewDeleteRecordWorker(logger, repo)
	in := worker.GetInChannel()

	service := URLService{
		repository: repo,
		resolver:   resolver,
		baseURL:    baseURL,
		ch:         in,
		logger:     logger,
	}

	go worker.FlushRecords()

	return &service
}

func (s *URLService) PingContext(ctx context.Context) error {
	return s.repository.PingContext(ctx)
}

func (s *URLService) CreateURLRecord(ctx context.Context, long string, userID string) (*storage.URLRecord, error) {
	shortURL := s.resolver.LongToShort(long)

	return s.repository.Write(ctx, storage.URLRecord{Original: long, Short: shortURL, UserID: userID})
}

func (s *URLService) DeleteURLRecords(ctx context.Context, rs []storage.URLRecord) {
	s.logger.Info("Sending to a delete channel")
	for _, record := range rs {
		s.ch <- record
	}
}

func (s *URLService) CreateURLRecords(ctx context.Context, rs []models.BatchRequest, userID string) (*[]models.BatchResponse, error) {
	var resultNew []models.BatchResponse

	if len(rs) != 0 {
		records := make([]storage.URLRecord, 0)

		for _, url := range rs {
			short := s.resolver.LongToShort(url.OriginalURL)

			records = append(records, storage.URLRecord{Original: url.OriginalURL, ID: url.CorrelationID, Short: short, UserID: userID})
		}
		err := s.repository.WriteAll(ctx, records)

		if err != nil {
			return &resultNew, err
		}

		for _, nr := range records {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: s.baseURL + "/" + nr.Short})
		}
	}

	return &resultNew, nil
}

func (s *URLService) GetURLByShort(ctx context.Context, short string) (*storage.URLRecord, error) {
	return s.repository.FindByShort(ctx, short)
}

func (s *URLService) GetURLByUserID(ctx context.Context, id string) (*[]models.ByIDRequest, error) {
	var resultNew []models.ByIDRequest

	urls, err := s.repository.FindByUserID(ctx, id)
	if err != nil {
		return &resultNew, err
	}

	if urls == nil {
		return &resultNew, err
	}

	for _, url := range *urls {
		resultNew = append(resultNew, models.ByIDRequest{ShortURL: s.baseURL + "/" + url.Short, OriginalURL: url.Original})
	}

	return &resultNew, nil
}
