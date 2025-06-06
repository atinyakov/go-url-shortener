package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func TestMemoryStorage_WriteAndFind(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	record := storage.URLRecord{
		Original: "https://example.com",
		Short:    "abc123",
		UserID:   "user1",
	}

	// Write
	result, err := mem.Write(context.Background(), record)
	assert.NoError(t, err)
	assert.Equal(t, record.Original, result.Original)

	// Write same short again - should fail
	_, err = mem.Write(context.Background(), record)
	assert.EqualError(t, err, "already exists")

	// Find by short
	found, err := mem.FindByShort(context.Background(), "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", found.Original)

	// Find non-existing short
	_, err = mem.FindByShort(context.Background(), "notfound")
	assert.EqualError(t, err, "not found")
}

func TestMemoryStorage_WriteAll(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	records := []storage.URLRecord{
		{Original: "https://1.com", Short: "s1", UserID: "u1"},
		{Original: "https://2.com", Short: "s2", UserID: "u1"},
	}

	err := mem.WriteAll(context.Background(), records)
	assert.NoError(t, err)

	// Ensure both are stored
	found1, _ := mem.FindByShort(context.Background(), "s1")
	found2, _ := mem.FindByShort(context.Background(), "s2")
	assert.Equal(t, "https://1.com", found1.Original)
	assert.Equal(t, "https://2.com", found2.Original)
}

func TestMemoryStorage_FindByUserID(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	mem.Write(context.Background(), storage.URLRecord{Short: "s1", Original: "https://a.com", UserID: "userX"})
	mem.Write(context.Background(), storage.URLRecord{Short: "s2", Original: "https://b.com", UserID: "userX"})

	records, err := mem.FindByUserID(context.Background(), "userX")
	assert.NoError(t, err)
	assert.Len(t, *records, 2)

	records, err = mem.FindByUserID(context.Background(), "unknown")
	assert.NoError(t, err)
	assert.Nil(t, records)
}

func TestMemoryStorage_GetStats(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	mem.Write(context.Background(), storage.URLRecord{Short: "s1", Original: "https://a.com", UserID: "userX"})
	mem.Write(context.Background(), storage.URLRecord{Short: "s2", Original: "https://b.com", UserID: "userX"})

	record, err := mem.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, record.Users, 1)
	assert.Equal(t, record.Urls, 2)
}

func TestMemoryStorage_Read(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	records, err := mem.Read(context.Background())
	assert.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestMemoryStorage_DeleteBatch(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	record := storage.URLRecord{
		ID:       "user1",
		UserID:   "user1",
		Original: "https://del.com",
		Short:    "toDel",
	}
	mem.Write(context.Background(), record)

	err := mem.DeleteBatch(context.Background(), []storage.URLRecord{record})
	assert.NoError(t, err)

	_, err = mem.FindByShort(context.Background(), "toDel")
	assert.EqualError(t, err, "not found")
}

func TestMemoryStorage_PingContext(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	err := mem.PingContext(context.Background())
	assert.True(t, errors.Is(err, errors.ErrUnsupported))
}

func TestMemoryStorage_FindByID(t *testing.T) {
	mem, _ := storage.CreateMemoryStorage()

	_, err := mem.FindByID(context.Background(), "nonexistent")
	assert.EqualError(t, err, "not found")
}
