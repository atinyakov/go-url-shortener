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
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	repo := CreateURLRepository(db, zap.NewNop())
	return db, mock, repo
}

func TestWrite(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	record := storage.URLRecord{
		Original: "https://example.com",
		Short:    "abc123",
		UserID:   "user-id-123",
	}

	mock.ExpectQuery(`INSERT INTO url_records`).
		WithArgs(record.Original, record.Short, "", record.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"original_url", "short_url", "id", "user_id"}).
			AddRow(record.Original, record.Short, "generated-uuid", record.UserID))

	result, err := repo.Write(context.Background(), record)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, record.Original, result.Original)
	assert.Equal(t, record.Short, result.Short)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRead(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	expectedRows := sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id"}).
		AddRow("id-1", "https://example.com", "abc123", "user-id-1").
		AddRow("id-2", "https://example2.com", "abc456", "user-id-2")

	mock.ExpectQuery(`SELECT \* FROM url_records;`).
		WillReturnRows(expectedRows)

	result, err := repo.Read(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "https://example.com", result[0].Original)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByShort(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	short := "abc123"
	expectedRecord := storage.URLRecord{
		ID:        "id-1",
		Original:  "https://example.com",
		Short:     short,
		UserID:    "user-id-1",
		IsDeleted: false,
	}

	mock.ExpectQuery(`SELECT id, original_url, short_url, user_id, is_deleted FROM url_records WHERE short_url = \$1;`).
		WithArgs(short).
		WillReturnRows(sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id", "is_deleted"}).
			AddRow(expectedRecord.ID, expectedRecord.Original, expectedRecord.Short, expectedRecord.UserID, expectedRecord.IsDeleted))

	result, err := repo.FindByShort(context.Background(), short)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedRecord.Original, result.Original)
	assert.Equal(t, expectedRecord.Short, result.Short)

	assert.NoError(t, mock.ExpectationsWereMet())
}
func TestFindByUserID(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	expectedUserID := "user-id-1"
	expectedRecord := storage.URLRecord{
		ID:       "id-1",
		Original: "https://example.com",
		Short:    "abc123",
		UserID:   expectedUserID,
	}

	mock.ExpectQuery(`SELECT id, original_url, short_url, user_id FROM url_records WHERE user_id = \$1;`).
		WithArgs(expectedUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id"}).
			AddRow(expectedRecord.ID, expectedRecord.Original, expectedRecord.Short, expectedRecord.UserID))

	result, err := repo.FindByUserID(context.Background(), expectedUserID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(*result), 0, "expected at least one record")
	assert.Equal(t, expectedRecord.Original, (*result)[0].Original)
	assert.Equal(t, expectedRecord.Short, (*result)[0].Short)
	assert.Equal(t, expectedRecord.UserID, (*result)[0].UserID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	id := "id-1"
	expected := storage.URLRecord{
		ID:       id,
		Original: "https://example.com",
		Short:    "short-url",
		UserID:   "user-id-1",
	}

	mock.ExpectQuery("SELECT \\* FROM url_records WHERE id = \\$1;").
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id"}).
			AddRow(expected.ID, expected.Original, expected.Short, expected.UserID))

	result, err := repo.FindByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestFindByLong(t *testing.T) {
	_, mock, repo := setupMockDB(t)

	short := "abc123"
	expected := storage.URLRecord{
		ID:        "id-1",
		Original:  "https://example.com",
		Short:     short,
		UserID:    "user-id-1",
		IsDeleted: false,
	}

	mock.ExpectQuery(`SELECT id, original_url, short_url, user_id, is_deleted FROM url_records WHERE short_url = \$1;`).
		WithArgs(short).
		WillReturnRows(sqlmock.NewRows([]string{"id", "original_url", "short_url", "user_id", "is_deleted"}).
			AddRow(expected.ID, expected.Original, expected.Short, expected.UserID, expected.IsDeleted))

	result, err := repo.FindByLong(context.Background(), short)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expected, *result)
}

func TestDeleteBatch(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	records := []storage.URLRecord{
		{Short: "short1", UserID: "user1"},
		{Short: "short2", UserID: "user1"},
	}

	mock.ExpectBegin()

	stmt := mock.ExpectPrepare("UPDATE url_records SET is_deleted = TRUE WHERE short_url = \\$1 AND user_id = \\$2")
	stmt.ExpectExec().WithArgs("short1", "user1").WillReturnResult(sqlmock.NewResult(1, 1))
	stmt.ExpectExec().WithArgs("short2", "user1").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := repo.DeleteBatch(context.Background(), records)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
