package service

import (
	"context"

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
