package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type StorageI interface {
	Write(value interface{}) error
	Read() ([]map[string]string, error)
}

type FileStorage struct {
	file *os.File
}

func (fs *FileStorage) Create(p string) (*FileStorage, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Open the file in read-write mode; create if not exists
	file, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}

	fs.file = file
	return fs, nil
}

func (fs *FileStorage) Write(value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = fs.file.Write(append(b, '\n'))
	return err
}

func (fs *FileStorage) Read() ([]map[string]string, error) {
	// Reset file pointer to the beginning
	_, err := fs.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	var records []map[string]string
	scanner := bufio.NewScanner(fs.file)
	for scanner.Scan() {
		line := scanner.Text()
		var record map[string]string
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
