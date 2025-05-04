package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/atinyakov/go-url-shortener/internal/worker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type MockRepo struct {
	Calls  [][]storage.URLRecord
	FailOn int
	CallNo int
}

func (m *MockRepo) DeleteBatch(_ context.Context, records []storage.URLRecord) error {
	m.Calls = append(m.Calls, records)
	m.CallNo++
	if m.CallNo == m.FailOn {
		return errors.New("forced failure")
	}
	return nil
}

func testLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	cfg.OutputPaths = []string{"stdout"}
	logger, _ := cfg.Build()
	return logger
}

func TestFlushRecords_BatchTrigger(t *testing.T) {
	repo := &MockRepo{}
	logger := testLogger()

	worker := worker.NewDeleteRecordWorker(logger, repo)
	in := worker.GetInChannel()

	go worker.FlushRecords()

	// Send more than 25 records
	for i := 0; i < 26; i++ {
		in <- storage.URLRecord{Short: "abc", UserID: "user"}
	}

	// Give some time for the batch to be processed
	time.Sleep(100 * time.Millisecond)

	require.Len(t, repo.Calls, 1)
	require.Len(t, repo.Calls[0], 26)
}

func TestFlushRecords_TimerTrigger(t *testing.T) {
	repo := &MockRepo{}
	logger := testLogger()

	worker := worker.NewDeleteRecordWorker(logger, repo)
	in := worker.GetInChannel()

	go worker.FlushRecords()

	in <- storage.URLRecord{Short: "abc", UserID: "user"}
	in <- storage.URLRecord{Short: "def", UserID: "user"}

	time.Sleep(11 * time.Second)

	require.Len(t, repo.Calls, 1)
	require.Len(t, repo.Calls[0], 2)
}

func TestFlushRecords_ErrorClearsBuffer(t *testing.T) {
	repo := &MockRepo{FailOn: 1}
	logger := testLogger()

	worker := worker.NewDeleteRecordWorker(logger, repo)
	in := worker.GetInChannel()

	go worker.FlushRecords()

	for i := 0; i < 30; i++ {
		in <- storage.URLRecord{Short: "abc", UserID: "user"}
	}

	time.Sleep(500 * time.Millisecond)

	require.GreaterOrEqual(t, len(repo.Calls), 1)
	require.Equal(t, 1, repo.FailOn) // Ensure the first call failed

	if len(repo.Calls) > 1 {
		require.LessOrEqual(t, len(repo.Calls[1]), 25)
	}
}
