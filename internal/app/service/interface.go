package service

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type Storage interface {
	Write(context.Context, storage.URLRecord) (*storage.URLRecord, error)
	WriteAll(context.Context, []storage.URLRecord) error
	Read(context.Context) ([]storage.URLRecord, error)
	DeleteBatch(context.Context, []storage.URLRecord) error
	FindByShort(context.Context, string) (*storage.URLRecord, error)
	FindByUserID(context.Context, string) (*[]storage.URLRecord, error)
	PingContext(context.Context) error
	FindByID(context.Context, string) (storage.URLRecord, error)
}

type URLServiceIface interface {
	CreateURLRecord(ctx context.Context, long string, userID string) (*storage.URLRecord, error)
	CreateURLRecords(ctx context.Context, rs []models.BatchRequest, userID string) (*[]models.BatchResponse, error)
	DeleteURLRecords(ctx context.Context, rs []storage.URLRecord)
	GetURLByShort(ctx context.Context, short string) (*storage.URLRecord, error)
	GetURLByUserID(ctx context.Context, id string) (*[]models.ByIDRequest, error)
	PingContext(ctx context.Context) error
}
