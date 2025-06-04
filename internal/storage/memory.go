// Package storage provides an in-memory implementation of a URL storage system.
// It supports writing, reading, deleting, and querying short/long URL pairs
// along with user-specific URL tracking. This is intended for testing and
// non-persistent scenarios.

package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/atinyakov/go-url-shortener/internal/models"
)

// MemoryStorage provides an in-memory store for URL records.
// It maps short URLs to original URLs and maintains per-user URL records.
// This implementation is concurrency-safe via sync.RWMutex.
type MemoryStorage struct {
	stol  map[string]string      // Maps short URL to original URL
	idtol map[string][]URLRecord // Maps user ID to their list of URLRecords
	mu    sync.RWMutex           // Guards access to the maps
}

// CreateMemoryStorage initializes and returns a new MemoryStorage instance.
func CreateMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		stol:  make(map[string]string),
		idtol: make(map[string][]URLRecord),
		mu:    sync.RWMutex{},
	}, nil
}

// Read returns all URL records in storage.
// In the in-memory implementation, this always returns an empty slice.
func (m *MemoryStorage) Read(ctx context.Context) ([]URLRecord, error) {
	return make([]URLRecord, 0), nil
}

// Write adds a new URLRecord to the memory storage.
// If the short URL already exists for the user, an error is returned.
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

// WriteAll writes multiple URLRecords in a batch.
// It stops and returns an error if any individual record fails to write.
func (m *MemoryStorage) WriteAll(ctx context.Context, records []URLRecord) error {
	for _, r := range records {
		_, e := m.Write(ctx, r)
		if e != nil {
			return e
		}
	}
	return nil
}

// FindByShort looks up a URLRecord by its short URL.
// Returns an error if the short URL is not found.
func (m *MemoryStorage) FindByShort(ctx context.Context, short string) (*URLRecord, error) {
	if long, exists := m.stol[short]; exists {
		return &URLRecord{
			Short:    short,
			Original: long,
		}, nil
	}
	return nil, errors.New("not found")
}

// DeleteBatch removes URL records from storage based on the given slice.
// In this implementation, only the mapping for the first ID is removed from idtol.
func (m *MemoryStorage) DeleteBatch(ctx context.Context, rs []URLRecord) error {
	delete(m.idtol, rs[0].ID)
	for _, r := range rs {
		delete(m.stol, r.Short)
	}
	return nil
}

// PingContext checks the storage connection health.
// For MemoryStorage, this returns an unsupported error.
func (m *MemoryStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}

// FindByUserID retrieves all URLRecords associated with a specific user ID.
func (m *MemoryStorage) FindByUserID(ctx context.Context, id string) (*[]URLRecord, error) {
	if items, exists := m.idtol[id]; exists {
		return &items, nil
	}
	return nil, nil
}

// FindByID returns a URLRecord by its ID.
// This method is not implemented and always returns an error.
func (m *MemoryStorage) FindByID(ctx context.Context, id string) (URLRecord, error) {
	return URLRecord{}, errors.New("not found")
}

// GetStats returns stats
func (m *MemoryStorage) GetStats(ctx context.Context) (*models.StatsResponse, error) {
	return &models.StatsResponse{
		Users: len(m.idtol),
		Urls:  len(m.stol),
	}, nil
}
