package storage

import "context"

type StorageI interface {
	Write(URLRecord) error
	Read() ([]URLRecord, error)
	FindByOriginal(string) (URLRecord, error)
	FindByShort(string) (URLRecord, error)
	PingContext(context.Context) error
}
