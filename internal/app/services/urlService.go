package services

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type URLService struct {
	repository Storage
}

func NewURLService(repo Storage) *URLService {
	return &URLService{
		repository: repo,
	}
}

func (s *URLService) PingContext(ctx context.Context) error {
	return s.repository.PingContext(ctx)
}

func (s *URLService) CreateURLRecord(r storage.URLRecord) (*storage.URLRecord, error) {
	return s.repository.Write(r)
}

func (s *URLService) CreateURLRecords(rs []storage.URLRecord) error {
	return s.repository.WriteAll(rs)
}

func (s *URLService) GetURLByShort(short string) (*storage.URLRecord, error) {
	return s.repository.FindByShort(short)
}
