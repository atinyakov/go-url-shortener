package storage

import (
	"context"
	"errors"
	"sync"
)

type MemoryStorage struct {
	stol map[string]string
	mu   sync.RWMutex
}

func CreateMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		stol: make(map[string]string),
		mu:   sync.RWMutex{},
	}, nil
}

func (m *MemoryStorage) Read() ([]URLRecord, error) {
	return make([]URLRecord, 0), nil
}

func (m *MemoryStorage) Write(record URLRecord) (*URLRecord, error) {
	long := record.Original
	short := record.Short
	m.mu.Lock()
	m.stol[short] = long
	m.mu.Unlock()
	return &record, nil
}

func (m *MemoryStorage) WriteAll(records []URLRecord) error {
	for _, r := range records {
		_, e := m.Write(r)
		if e != nil {
			return e
		}
	}
	return nil
}

func (m *MemoryStorage) FindByShort(short string) (*URLRecord, error) {
	if long, exists := m.stol[short]; exists {
		return &URLRecord{
			Short:    short,
			Original: long,
		}, nil
	}
	return nil, errors.New("not found")
}

func (m *MemoryStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}

func (m *MemoryStorage) FindByID(id string) (URLRecord, error) {
	return URLRecord{}, errors.New("not found")
}
