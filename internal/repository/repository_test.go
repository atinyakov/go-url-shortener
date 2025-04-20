package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Helper to set up a mock DB and repository
func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *URLRepository) {
	// Create a mock DB
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Create the repository instance
	repo := CreateURLRepository(db, zap.NewNop()) // Use a no-op logger for simplicity
	return db, mock, repo
}

func TestWrite(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	// Define the input and expected output
	record := storage.URLRecord{
		Original: "https://example.com",
		Short:    "abc123",
		UserID:   "user-id-123",
	}

	// Set up the mock to expect the INSERT query and return the inserted record
	mock.ExpectQuery(`INSERT INTO url_records`).
		WithArgs(record.Original, record.Short, "", record.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"original_url", "short_url", "id", "user_id"}).
			AddRow(record.Original, record.Short, "generated-uuid", record.UserID))

	// Call the Write method
	result, err := repo.Write(context.Background(), record)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, record.Original, result.Original)
	assert.Equal(t, record.Short, result.Short)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRead(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	// Define the expected rows
	expectedRows := sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id"}).
		AddRow("id-1", "https://example.com", "abc123", "user-id-1").
		AddRow("id-2", "https://example2.com", "abc456", "user-id-2")

	// Set up the mock to return the expected rows for the SELECT query
	mock.ExpectQuery(`SELECT \* FROM url_records;`).
		WillReturnRows(expectedRows)

	// Call the Read method
	result, err := repo.Read(context.Background())

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "https://example.com", result[0].Original)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByShort(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	// Define the input short URL and expected result
	short := "abc123"
	expectedRecord := storage.URLRecord{
		ID:        "id-1",
		Original:  "https://example.com",
		Short:     short,
		UserID:    "user-id-1",
		IsDeleted: false,
	}

	// Set up the mock to return the expected row for the short URL
	mock.ExpectQuery(`SELECT id, original_url, short_url, user_id, is_deleted FROM url_records WHERE short_url = \$1;`).
		WithArgs(short).
		WillReturnRows(sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id", "is_deleted"}).
			AddRow(expectedRecord.ID, expectedRecord.Original, expectedRecord.Short, expectedRecord.UserID, expectedRecord.IsDeleted))

	// Call the FindByShort method
	result, err := repo.FindByShort(context.Background(), short)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedRecord.Original, result.Original)
	assert.Equal(t, expectedRecord.Short, result.Short)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
