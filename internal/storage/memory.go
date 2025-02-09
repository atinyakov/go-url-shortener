package storage

import (
	"context"
	"errors"
)

type MemoryStorage struct {
	ltos map[string]string
	stol map[string]string
}

func CreateMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		ltos: make(map[string]string),
		stol: make(map[string]string),
	}, nil
}

func (m *MemoryStorage) Read() ([]URLRecord, error) {
	return make([]URLRecord, 0), nil
}

func (m *MemoryStorage) Write(record URLRecord) error {
	long := record.Original
	short := record.Short

	m.ltos[long] = short
	m.stol[short] = long
	return nil
}

func (m *MemoryStorage) FindByShort(short string) (URLRecord, error) {
	if long, exists := m.stol[short]; exists {
		return URLRecord{
			Short:    short,
			Original: long,
		}, nil
	}
	return URLRecord{}, errors.New("not found")
}

func (m *MemoryStorage) FindByOriginal(long string) (URLRecord, error) {
	if short, exists := m.ltos[long]; exists {
		return URLRecord{
			Short:    short,
			Original: long,
		}, nil
	}

	return URLRecord{}, errors.New("not found")
}

func (m *MemoryStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}
