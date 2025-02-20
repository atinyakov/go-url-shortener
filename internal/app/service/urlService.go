package service

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type URLService struct {
	repository Storage
	resolver   *URLResolver
	baseURL    string
}

func NewURL(repo Storage, resolver *URLResolver, baseURL string) *URLService {
	return &URLService{
		repository: repo,
		resolver:   resolver,
		baseURL:    baseURL,
	}
}

func (s *URLService) PingContext(ctx context.Context) error {
	return s.repository.PingContext(ctx)
}

func (s *URLService) CreateURLRecord(long string, userID string) (*storage.URLRecord, error) {
	shortURL := s.resolver.LongToShort(long)

	return s.repository.Write(storage.URLRecord{Original: long, Short: shortURL, UserID: userID})
}

func (s *URLService) CreateURLRecords(rs []models.BatchRequest, userID string) (*[]models.BatchResponse, error) {
	var resultNew []models.BatchResponse

	if len(rs) != 0 {
		records := make([]storage.URLRecord, 0)

		for _, url := range rs {
			short := s.resolver.LongToShort(url.OriginalURL)

			records = append(records, storage.URLRecord{Original: url.OriginalURL, ID: url.CorrelationID, Short: short, UserID: userID})
		}
		err := s.repository.WriteAll(records)

		if err != nil {
			return &resultNew, err
		}

		for _, nr := range records {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: s.baseURL + "/" + nr.Short})
		}
	}

	return &resultNew, nil
}

func (s *URLService) GetURLByShort(short string) (*storage.URLRecord, error) {
	return s.repository.FindByShort(short)
}

func (s *URLService) GetURLByUserID(id string) (*[]models.ByIDRequest, error) {
	var resultNew []models.ByIDRequest

	urls, err := s.repository.FindByUserID(id)
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
