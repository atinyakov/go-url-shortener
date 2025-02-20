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

func (fs *FileStorage) Write(value URLRecord) (*URLRecord, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	encoder := json.NewEncoder(fs.file)
	return &value, encoder.Encode(value)
}

func (fs *FileStorage) WriteAll(records []URLRecord) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, r := range records {
		if _, err := fs.Write(r); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileStorage) Read() ([]URLRecord, error) {
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

func (fs *FileStorage) FindByShort(s string) (*URLRecord, error) {

	fs.logger.Info("Got short:", zap.String("shortUrl", s))
	records, err := fs.Read()
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

func (fs *FileStorage) FindByID(id string) (URLRecord, error) {
	records, err := fs.Read()
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

func (fs *FileStorage) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

func (fs *FileStorage) PingContext(c context.Context) error {
	return errors.ErrUnsupported
}
