package storage

import "context"

type StorageI interface {
	Write(URLRecord) error
	WriteAll([]URLRecord) error
	Read() ([]URLRecord, error)
	FindByShort(string) (URLRecord, error)
	PingContext(context.Context) error
	FindByID(string) (URLRecord, error)
}
