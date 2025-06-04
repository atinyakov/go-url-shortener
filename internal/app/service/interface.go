// Package service provides interfaces and methods for interacting with URL storage and management.
// It includes methods for creating, reading, updating, and deleting URL records, as well as pinging the storage backend.
package service

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
)

// Storage is an interface for interacting with the underlying storage backend for URL records.
// It provides methods for reading, writing, and deleting URL records, as well as querying based on short URL and user ID.
type Storage interface {
	// Write adds a single URL record to the storage.
	Write(context.Context, storage.URLRecord) (*storage.URLRecord, error)

	// WriteAll adds multiple URL records to the storage.
	WriteAll(context.Context, []storage.URLRecord) error

	// Read retrieves all URL records from the storage.
	Read(context.Context) ([]storage.URLRecord, error)

	// DeleteBatch deletes multiple URL records from the storage.
	DeleteBatch(context.Context, []storage.URLRecord) error

	// FindByShort retrieves a URL record by its shortened URL.
	FindByShort(context.Context, string) (*storage.URLRecord, error)

	// FindByUserID retrieves all URL records associated with a given user ID.
	FindByUserID(context.Context, string) (*[]storage.URLRecord, error)

	// PingContext checks the connectivity to the storage backend.
	PingContext(context.Context) error

	// FindByID retrieves a URL record by its ID.
	FindByID(context.Context, string) (storage.URLRecord, error)

	// GetStats returns number of users and number of shortend URLs
	GetStats(context.Context) (*models.StatsResponse, error)
}

// URLServiceIface is an interface that defines the URL service's core functionality.
// It allows for URL record creation, retrieval, deletion, and service health checks.
type URLServiceIface interface {
	// CreateURLRecord creates a new URL record based on a long URL and user ID.
	CreateURLRecord(ctx context.Context, long string, userID string) (*storage.URLRecord, error)

	// CreateURLRecords creates multiple URL records in batch, based on a list of requests and user ID.
	CreateURLRecords(ctx context.Context, rs []models.BatchRequest, userID string) (*[]models.BatchResponse, error)

	// DeleteURLRecords deletes multiple URL records in batch.
	DeleteURLRecords(ctx context.Context, rs []storage.URLRecord)

	// GetURLByShort retrieves a URL record by its shortened URL.
	GetURLByShort(ctx context.Context, short string) (*storage.URLRecord, error)

	// GetURLByUserID retrieves all URL records associated with a given user ID.
	GetURLByUserID(ctx context.Context, id string) (*[]models.ByIDRequest, error)

	// PingContext checks the health of the URL service.
	PingContext(ctx context.Context) error

	// GetStats retrieves stats for all URL records.
	GetStats(ctx context.Context) (*models.StatsResponse, error)
}
