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

	"go.uber.org/zap"
)

type FileStorage struct {
	file   *os.File
	mu     sync.RWMutex
	logger *zap.Logger
}

func NewFileStorage(p string, logger *zap.Logger) (*FileStorage, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Open the file in read-write mode; create if not exists
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

func (fs *FileStorage) Write(ctx context.Context, value URLRecord) (*URLRecord, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	encoder := json.NewEncoder(fs.file)
	return &value, encoder.Encode(value)
}

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

func (fs *FileStorage) Read(ctx context.Context) ([]URLRecord, error) {
	// Reset file pointer to the beginning
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

func (fs *FileStorage) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

func (fs *FileStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}
