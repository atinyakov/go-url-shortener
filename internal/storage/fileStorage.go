package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileStorage struct {
	file *os.File
	mu   sync.RWMutex
}

func NewFileStorage(p string) (*FileStorage, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Open the file in read-write mode; create if not exists
	file, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		file: file,
		mu:   sync.RWMutex{},
	}, nil
}

type URLRecord struct {
	ID       string `json:"uuid" format:"uuid"`
	Original string `json:"original_url"`
	Short    string `json:"short_url"`
}

func (fs *FileStorage) Write(value URLRecord) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	encoder := json.NewEncoder(fs.file)
	return encoder.Encode(value)
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

func (fs *FileStorage) Close() error {
	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}
