package services

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type Storage interface {
	Write(storage.URLRecord) (*storage.URLRecord, error)
	WriteAll([]storage.URLRecord) error
	Read() ([]storage.URLRecord, error)
	FindByShort(string) (*storage.URLRecord, error)
	PingContext(context.Context) error
	FindByID(string) (storage.URLRecord, error)
}
