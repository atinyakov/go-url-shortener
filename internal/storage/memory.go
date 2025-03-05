package storage

import (
	"context"
	"errors"
	"sync"
)

type MemoryStorage struct {
	stol  map[string]string
	idtol map[string][]URLRecord
	mu    sync.RWMutex
}

func CreateMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		stol:  make(map[string]string),
		idtol: make(map[string][]URLRecord),
		mu:    sync.RWMutex{},
	}, nil
}

func (m *MemoryStorage) Read(ctx context.Context) ([]URLRecord, error) {
	return make([]URLRecord, 0), nil
}

func (m *MemoryStorage) Write(ctx context.Context, record URLRecord) (*URLRecord, error) {
	long := record.Original
	short := record.Short

	existingURLs := m.idtol[record.UserID]
	if len(existingURLs) > 0 {
		for _, url := range existingURLs {
			if url.Short == record.Short {
				return nil, errors.New("already exists")
			}
		}
	}
	m.mu.Lock()

	m.idtol[record.UserID] = append(m.idtol[record.UserID], record)
	m.stol[short] = long

	m.mu.Unlock()
	return &record, nil
}

func (m *MemoryStorage) WriteAll(ctx context.Context, records []URLRecord) error {
	for _, r := range records {
		_, e := m.Write(ctx, r)
		if e != nil {
			return e
		}
	}
	return nil
}

func (m *MemoryStorage) FindByShort(ctx context.Context, short string) (*URLRecord, error) {
	if long, exists := m.stol[short]; exists {
		return &URLRecord{
			Short:    short,
			Original: long,
		}, nil
	}
	return nil, errors.New("not found")
}

func (m *MemoryStorage) DeleteBatch(ctx context.Context, rs []URLRecord) error {
	delete(m.idtol, rs[0].ID)
	for _, r := range rs {
		delete(m.stol, r.Short)
	}
	return nil
}

func (m *MemoryStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}

func (m *MemoryStorage) FindByUserID(ctx context.Context, id string) (*[]URLRecord, error) {
	if items, exists := m.idtol[id]; exists {
		return &items, nil
	}
	return nil, nil
}

func (m *MemoryStorage) FindByID(ctx context.Context, id string) (URLRecord, error) {
	return URLRecord{}, errors.New("not found")
}
