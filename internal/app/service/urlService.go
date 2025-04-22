// Package service provides the URL service functionality, including operations
// for creating, deleting, and retrieving short and long URLs. It interacts
// with the storage backend, URL resolver, and worker processes for background
// operations like deleting URL records.
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/models"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/atinyakov/go-url-shortener/internal/worker"
)

// URLService is responsible for providing the main URL-related services,
// including URL creation, deletion, and retrieval. It interacts with the
// underlying storage and resolver, and uses a worker for background tasks.
type URLService struct {
	// repository is the storage interface for interacting with the URL data.
	repository Storage
	// resolver is responsible for converting long URLs to short URLs.
	resolver *URLResolver
	// logger is used for logging operations within the service.
	logger *zap.Logger
	// baseURL is the base URL for constructing the full short URL.
	baseURL string
	// ch is a channel used to send URL records for deletion to a worker.
	ch chan<- storage.URLRecord
}

// NewURL creates a new instance of URLService with the given repository, resolver,
// logger, and base URL. It initializes the worker for background deletion tasks.
func NewURL(repo Storage, resolver *URLResolver, logger *zap.Logger, baseURL string) *URLService {
	// Initialize the delete worker
	worker := worker.NewDeleteRecordWorker(logger, repo)
	in := worker.GetInChannel()

	// Create the URLService and start the worker's background operation
	service := URLService{
		repository: repo,
		resolver:   resolver,
		baseURL:    baseURL,
		ch:         in,
		logger:     logger,
	}

	// Start the worker to process URL deletions
	go worker.FlushRecords()

	return &service
}

// PingContext checks the health of the storage connection.
func (s *URLService) PingContext(ctx context.Context) error {
	// Ping the repository to ensure it is accessible
	return s.repository.PingContext(ctx)
}

// CreateURLRecord creates a new URL record in the storage, generating a short URL
// from the provided long URL and associating it with the specified user ID.
func (s *URLService) CreateURLRecord(ctx context.Context, long string, userID string) (*storage.URLRecord, error) {
	// Generate a short URL using the resolver
	shortURL := s.resolver.LongToShort(long)

	// Store the URL record in the repository
	return s.repository.Write(ctx, storage.URLRecord{Original: long, Short: shortURL, UserID: userID})
}

// DeleteURLRecords sends URL records to the worker's channel for deletion.
// This will be processed asynchronously by the worker.
func (s *URLService) DeleteURLRecords(ctx context.Context, rs []storage.URLRecord) {
	// Log the deletion action and send each URL record to the worker for deletion
	s.logger.Info("Sending to a delete channel")
	for _, record := range rs {
		s.ch <- record
	}
}

// CreateURLRecords processes a batch of URL creation requests. It generates short URLs
// for the provided long URLs, stores them in the repository, and returns the batch response
// with the corresponding short URLs.
func (s *URLService) CreateURLRecords(ctx context.Context, rs []models.BatchRequest, userID string) (*[]models.BatchResponse, error) {
	var resultNew []models.BatchResponse

	if len(rs) != 0 {
		// Prepare the list of URL records to be created
		records := make([]storage.URLRecord, 0)

		// Generate short URLs for each request
		for _, url := range rs {
			short := s.resolver.LongToShort(url.OriginalURL)
			records = append(records, storage.URLRecord{Original: url.OriginalURL, ID: url.CorrelationID, Short: short, UserID: userID})
		}

		// Write all records to the repository
		err := s.repository.WriteAll(ctx, records)
		if err != nil {
			return &resultNew, err
		}

		// Build the response with the short URLs
		for _, nr := range records {
			resultNew = append(resultNew, models.BatchResponse{CorrelationID: nr.ID, ShortURL: s.baseURL + "/" + nr.Short})
		}
	}

	return &resultNew, nil
}

// GetURLByShort retrieves the original URL by the given short URL.
func (s *URLService) GetURLByShort(ctx context.Context, short string) (*storage.URLRecord, error) {
	// Find and return the URL record based on the short URL
	return s.repository.FindByShort(ctx, short)
}

// GetURLByUserID retrieves all URL records associated with the specified user ID.
func (s *URLService) GetURLByUserID(ctx context.Context, id string) (*[]models.ByIDRequest, error) {
	var resultNew []models.ByIDRequest

	// Retrieve the URL records from the repository based on the user ID
	urls, err := s.repository.FindByUserID(ctx, id)
	if err != nil {
		return &resultNew, err
	}

	// If no URLs found, return an empty result
	if urls == nil {
		return &resultNew, err
	}

	// Build the response with the full URLs (including base URL)
	for _, url := range *urls {
		resultNew = append(resultNew, models.ByIDRequest{ShortURL: s.baseURL + "/" + url.Short, OriginalURL: url.Original})
	}

	return &resultNew, nil
}
