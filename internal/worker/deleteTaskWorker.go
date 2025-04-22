// Package worker provides background workers for asynchronous processing tasks,
// such as batched deletion of URL records.
package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

// Repo is an interface that defines a method for batch-deleting URL records.
// It is used to decouple the worker from a specific storage implementation.
type Repo interface {
	DeleteBatch(context.Context, []storage.URLRecord) error
}

// DeleteTaskWorker is a background worker responsible for collecting and
// deleting URL records in batches. It accepts records through a channel and
// periodically flushes them to the storage.
type DeleteTaskWorker struct {
	in     chan storage.URLRecord // Channel for incoming URL records to be deleted
	logger *zap.Logger            // Structured logger for debugging and error reporting
	repo   Repo                   // Storage layer interface for deletion
}

// NewDeleteRecordWorker creates and returns a new DeleteTaskWorker.
// It initializes the input channel and sets the logger and storage repository.
func NewDeleteRecordWorker(logger *zap.Logger, repo Repo) *DeleteTaskWorker {
	ch := make(chan storage.URLRecord)

	return &DeleteTaskWorker{
		in:     ch,
		logger: logger,
		repo:   repo,
	}
}

// GetInChannel returns a write-only channel for sending records to be deleted.
// Callers can push records to this channel to schedule them for deletion.
func (s *DeleteTaskWorker) GetInChannel() chan<- storage.URLRecord {
	s.logger.Info("get in channel")
	return s.in
}

// FlushRecords starts an infinite loop that receives records from the input
// channel and flushes them to the storage in batches. Records are sent either
// when the buffer reaches 25 items or every 10 seconds.
func (s *DeleteTaskWorker) FlushRecords() {
	s.logger.Info("Flushing records init")
	ticker := time.NewTicker(10 * time.Second)
	var messages []storage.URLRecord

	// sendMessages flushes the current batch of records to the repository.
	sendMessages := func() {
		s.logger.Info("Flushing delete records", zap.Int("count", len(messages)))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := s.repo.DeleteBatch(ctx, messages)
		if err != nil {
			s.logger.Error("Cannot delete records", zap.Error(err))
			messages = messages[:0] // clear buffer even on error
			return
		}
		messages = messages[:0] // clear buffer after success
	}

	for {
		select {
		case msg := <-s.in:
			s.logger.Info("Got record to delete", zap.Any("msg", msg))
			messages = append(messages, msg)
			if len(messages) > 25 {
				sendMessages()
			}
		case <-ticker.C:
			if len(messages) == 0 {
				continue
			}
			sendMessages()
		}
	}
}
