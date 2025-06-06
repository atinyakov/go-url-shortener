package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWrite(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tempFile := filepath.Join(t.TempDir(), "write_test.json")
	fs, err := NewFileStorage(tempFile, logger)
	if err != nil {
		t.Fatalf("failed to create FileStorage: %v", err)
	}
	defer fs.Close()

	record := URLRecord{
		Short:    "shorturl",
		Original: "https://example.com",
	}

	_, err = fs.Write(context.Background(), record)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestWriteAll(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "write_all_test.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	records := []URLRecord{
		{ID: "1", Short: "short1", UserID: "user1", Original: "http://example.com/1"},
		{ID: "2", Short: "short2", UserID: "user1", Original: "http://example.com/2"},
	}

	if err := fs.WriteAll(context.Background(), records); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	writtenRecords, err := fs.Read(context.Background())
	if err != nil {
		t.Fatalf("failed to read from file: %v", err)
	}

	if len(writtenRecords) != len(records) {
		t.Fatalf("expected %d records, got %d", len(records), len(writtenRecords))
	}

}

func TestFindByShort(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "find_by_short.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()
	records := []URLRecord{
		{Original: "https://example1.com", Short: "abc123", UserID: "user-id-1"},
		{Original: "https://example2.com", Short: "def456", UserID: "user-id-2"},
	}

	// Write records to storage
	err = fs.WriteAll(context.Background(), records)
	require.NoError(t, err)

	// Find record by short URL
	result, err := fs.FindByShort(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "https://example1.com", result.Original)
}

func TestFindByID(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "find_by_id.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	records := []URLRecord{
		{ID: "id-1", Original: "https://example1.com", Short: "abc123", UserID: "user-id-1"},
		{ID: "id-2", Original: "https://example2.com", Short: "def456", UserID: "user-id-2"},
	}

	// Write records to storage
	err = fs.WriteAll(context.Background(), records)
	require.NoError(t, err)

	// Find record by ID
	result, err := fs.FindByID(context.Background(), "id-1")
	require.NoError(t, err)
	assert.Equal(t, "https://example1.com", result.Original)
}

func TestFindByUserID(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "find_by_user_id.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	records := []URLRecord{
		{Original: "https://example1.com", Short: "abc123", UserID: "user-id-1"},
		{Original: "https://example2.com", Short: "def456", UserID: "user-id-1"},
		{Original: "https://example3.com", Short: "ghi789", UserID: "user-id-2"},
	}

	// Write records to storage
	err = fs.WriteAll(context.Background(), records)
	require.NoError(t, err)

	// Find records by user ID
	result, err := fs.FindByUserID(context.Background(), "user-id-1")
	require.NoError(t, err)
	assert.Len(t, *result, 2)
}

func TestDeleteBatch(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "test_delete_batch.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	records := []URLRecord{
		{Original: "https://example1.com", Short: "abc123", UserID: "user-id-1"},
		{Original: "https://example2.com", Short: "def456", UserID: "user-id-2"},
		{Original: "https://example3.com", Short: "ghi789", UserID: "user-id-1"},
	}

	// Write records to storage
	err = fs.WriteAll(context.Background(), records)
	require.NoError(t, err)

	// Delete batch
	err = fs.DeleteBatch(context.Background(), []URLRecord{
		{Short: "abc123"},
		{Short: "ghi789"},
	})
	require.NoError(t, err)

	// Read back the records
	remainingRecords, err := fs.Read(context.Background())
	require.NoError(t, err)
	assert.Len(t, remainingRecords, 1)
	assert.Equal(t, "https://example2.com", remainingRecords[0].Original)
}

func TestClose(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "test_close.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	// Close the storage
	err = fs.Close()
	require.NoError(t, err)
}

func TestPingContext(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "test_ping.json")

	fs, err := NewFileStorage(testFile, logger)
	if err != nil {
		t.Fatalf("could not create file storage: %v", err)
	}
	defer func() {
		fs.Close()
		os.Remove(testFile)
	}()

	// Call PingContext (which currently returns ErrUnsupported)
	err = fs.PingContext(context.Background())
	assert.Error(t, err)
}

func TestFileStorage_GetStats(t *testing.T) {
	logger, _ := zap.NewProduction()
	testFile := filepath.Join(os.TempDir(), "test_ping.json")

	fs, err := NewFileStorage(testFile, logger)

	assert.NoError(t, err)
	defer fs.Close()

	records := []URLRecord{
		{ID: "1", Short: "s1", Original: "https://a.com", UserID: "user1"},
		{ID: "2", Short: "s2", Original: "https://b.com", UserID: "user2"},
		{ID: "3", Short: "s3", Original: "https://c.com", UserID: "user1"},
		{ID: "4", Short: "s4", Original: "https://d.com", UserID: ""},
	}

	err = fs.WriteAll(context.Background(), records)
	assert.NoError(t, err)

	stats, err := fs.GetStats(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, 4, stats.Urls)  // total URLs
	assert.Equal(t, 2, stats.Users) // user1 and user2 only
}
