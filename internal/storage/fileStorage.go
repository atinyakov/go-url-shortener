// Package storage provides storage implementations for URL records,
// including an implementation backed by a local file using JSON encoding.
package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"go.uber.org/zap"
)

// FileStorage provides a file-based implementation of persistent storage
// for URL records. Each record is stored as a JSON-encoded line.
type FileStorage struct {
	file   *os.File     // Underlying file used for storage
	mu     sync.RWMutex // Mutex to protect concurrent access
	logger *zap.Logger  // Logger for internal debugging and error tracking
}

// NewFileStorage creates a new instance of FileStorage using the provided
// file path and logger. The file will be created if it does not exist.
func NewFileStorage(p string, logger *zap.Logger) (*FileStorage, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		file:   file,
		mu:     sync.RWMutex{},
		logger: logger,
	}, nil
}

// Write appends a single URLRecord to the file in JSON format.
func (fs *FileStorage) Write(ctx context.Context, value URLRecord) (*URLRecord, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	encoder := json.NewEncoder(fs.file)
	return &value, encoder.Encode(value)
}

// WriteAll overwrites the file with the provided slice of URLRecords,
// replacing all existing data.
func (fs *FileStorage) WriteAll(ctx context.Context, records []URLRecord) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := fs.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to beginning of file: %w", err)
	}

	writer := bufio.NewWriter(fs.file)

	for _, r := range records {
		if err := json.NewEncoder(writer).Encode(r); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffered writer: %w", err)
	}

	return nil
}

// Read parses all records from the file and returns them as a slice.
func (fs *FileStorage) Read(ctx context.Context) ([]URLRecord, error) {
	_, err := fs.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	var records []URLRecord
	scanner := bufio.NewScanner(fs.file)
	for scanner.Scan() {
		line := scanner.Text()
		var record URLRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("failed to parse JSON line: %w", err)
		}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return records, nil
}

// FindByShort searches for a URLRecord by its short URL value.
func (fs *FileStorage) FindByShort(ctx context.Context, s string) (*URLRecord, error) {
	fs.logger.Info("Got short:", zap.String("shortUrl", s))
	records, err := fs.Read(ctx)
	if err != nil {
		fs.logger.Error("FindByShort error=", zap.String("error", err.Error()))
		return nil, err
	}

	for _, r := range records {
		if r.Short == s {
			return &r, nil
		}
	}

	return nil, errors.New("not found")
}

// FindByID looks up a URLRecord by its unique ID field.
func (fs *FileStorage) FindByID(ctx context.Context, id string) (URLRecord, error) {
	records, err := fs.Read(ctx)
	if err != nil {
		return URLRecord{}, err
	}

	for _, r := range records {
		if r.ID == id {
			return r, nil
		}
	}

	return URLRecord{}, nil
}

// FindByUserID retrieves all records associated with a given user ID.
func (fs *FileStorage) FindByUserID(ctx context.Context, userID string) (*[]URLRecord, error) {
	records, err := fs.Read(ctx)
	res := make([]URLRecord, 0)

	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.UserID == userID {
			res = append(res, r)
		}
	}

	return &res, nil
}

// DeleteBatch removes all records from the file that match the short URLs
// of the provided slice of URLRecords.
func (fs *FileStorage) DeleteBatch(ctx context.Context, rs []URLRecord) error {
	records, err := fs.Read(ctx)
	if err != nil {
		return err
	}

	toDelete := make(map[string]struct{}, len(rs))
	for _, url := range rs {
		toDelete[url.Short] = struct{}{}
	}

	newRecords := make([]URLRecord, 0, len(records))
	for _, r := range records {
		if _, found := toDelete[r.Short]; !found {
			newRecords = append(newRecords, r)
		}
	}

	return fs.WriteAll(ctx, newRecords)
}

// Close closes the underlying file handle used by FileStorage.
func (fs *FileStorage) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// PingContext returns an error indicating that ping is unsupported
// in the file-based storage implementation.
func (fs *FileStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}

// GetStats returns stats
func (fs *FileStorage) GetStats(c context.Context) (*models.StatsResponse, error) {
	records, err := fs.Read(c)
	if err != nil {
		return nil, err
	}

	return &models.StatsResponse{
		Urls: len(records),
	}, nil
}
