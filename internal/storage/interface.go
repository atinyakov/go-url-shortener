package storage

type StorageI interface {
	Write(URLRecord) error
	Read() ([]URLRecord, error)
}
