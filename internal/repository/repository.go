package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

var ErrConflict = errors.New("data conflict")

func InitDB(ps string, logger *zap.Logger) *sql.DB {
	db, err := sql.Open("pgx", ps)
	if err != nil {
		logger.Fatal(err.Error())
	}

	if err := db.Ping(); err != nil {
		logger.Fatal(err.Error())
	}

	// Create table if not exists
	createTable := `
		CREATE TABLE IF NOT EXISTS url_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		original_url TEXT UNIQUE NOT NULL,
		short_url TEXT UNIQUE NOT NULL,
		is_deleted BOOLEAN DEFAULT FALSE,
		user_id UUID);`

	_, err = db.Exec(createTable)
	if err != nil {
		logger.Fatal(err.Error())
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS created_by ON url_records (user_id)")
	if err != nil {
		logger.Fatal(err.Error())
	}

	return db
}

type URLRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func CreateURLRepository(db *sql.DB, l *zap.Logger) *URLRepository {
	return &URLRepository{
		db:     db,
		logger: l,
	}
}

func (r *URLRepository) Write(ctx context.Context, v storage.URLRecord) (*storage.URLRecord, error) {
	var existing = v

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO url_records(original_url, short_url, id, user_id) 
		 VALUES ($1, $2, COALESCE(NULLIF($3, '')::UUID, gen_random_uuid()), $4)
		 ON CONFLICT (original_url) DO NOTHING 
		 RETURNING original_url, short_url, id, user_id;`,
		v.Original, v.Short, v.ID, v.UserID,
	).Scan(&existing.Original, &existing.Short, &existing.ID, &existing.UserID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &existing, ErrConflict
		}
		r.logger.Error("Write error=, while INSERT", zap.String("error", err.Error()))
		return nil, err
	}

	r.logger.Info("Insert successful!")
	return &existing, nil
}

func (r *URLRepository) WriteAll(ctx context.Context, rs []storage.URLRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		err := tx.Rollback()
		if err != nil {
			r.logger.Error("ROLLBACK error=", zap.String("error", err.Error()))
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO url_records(original_url, short_url, id, user_id) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (original_url) DO NOTHING 
		RETURNING original_url, short_url, id, user_id;
	`)
	if err != nil {
		return err
	}

	for _, v := range rs {

		defer stmt.Close()
		_, err = stmt.ExecContext(ctx, v.Original, v.Short, v.ID, v.UserID)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return ErrConflict
			}
			return err
		}
	}

	return tx.Commit()
}

func (r *URLRepository) Read(ctx context.Context) ([]storage.URLRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM url_records;")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	records := make([]storage.URLRecord, 0)

	for rows.Next() {
		var r storage.URLRecord
		err = rows.Scan(&r.ID, &r.Original, &r.Short, &r.UserID)
		if err != nil {
			return nil, err
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil

}

func (r *URLRepository) FindByShort(ctx context.Context, s string) (*storage.URLRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, original_url, short_url, user_id, is_deleted 
	FROM url_records WHERE short_url = $1;`, s)

	var id, originalURL, shortURL, userID string
	var IsDeleted bool

	err := row.Scan(&id, &originalURL, &shortURL, &userID, &IsDeleted)
	if err != nil {
		r.logger.Error("FindByShort err=", zap.String("error", err.Error()))
		return nil, err
	}

	return &storage.URLRecord{
		ID:        id,
		Original:  originalURL,
		Short:     shortURL,
		UserID:    userID,
		IsDeleted: IsDeleted,
	}, nil
}

func (r *URLRepository) DeleteBatch(ctx context.Context, rs []storage.URLRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		err := tx.Rollback()
		if err != nil {
			r.logger.Error("ROLLBACK error=", zap.String("error", err.Error()))
		}
	}()

	stmt, err := tx.Prepare(`
		UPDATE url_records 
		SET is_deleted = TRUE 
		WHERE short_url = $1 AND user_id = $2
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()
	for _, v := range rs {
		_, err = stmt.ExecContext(ctx, v.Short, v.UserID)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *URLRepository) FindByLong(ctx context.Context, long string) (*storage.URLRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, original_url, short_url, user_id, is_deleted 
	FROM url_records WHERE short_url = $1;`, long)

	var id, originalURL, shortURL, userID string
	var IsDeleted bool

	err := row.Scan(&id, &originalURL, &shortURL, &userID, &IsDeleted)
	if err != nil {
		r.logger.Error("FindByLong err=", zap.String("error", err.Error()))
		return nil, err
	}

	return &storage.URLRecord{
		ID:        id,
		Original:  originalURL,
		Short:     shortURL,
		UserID:    userID,
		IsDeleted: IsDeleted,
	}, nil
}

func (r *URLRepository) FindByID(ctx context.Context, s string) (storage.URLRecord, error) {
	row := r.db.QueryRowContext(ctx, "SELECT * FROM url_records WHERE id = $1;", s)

	var id, original, short, userID string

	err := row.Scan(&id, &original, &short, &userID)
	if err != nil {
		r.logger.Error("FindByID error=", zap.String("error", err.Error()))
		return storage.URLRecord{}, nil
	}

	return storage.URLRecord{
		ID:       id,
		Original: original,
		Short:    short,
		UserID:   userID,
	}, nil
}

func (r *URLRepository) FindByUserID(ctx context.Context, userID string) (*[]storage.URLRecord, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, original_url, short_url, user_id FROM url_records WHERE user_id = $1;", userID)
	if err != nil {
		r.logger.Error(fmt.Sprintf("FindByUserID error=%s", err.Error()))
		return &[]storage.URLRecord{}, nil
	}
	defer rows.Close()

	res := make([]storage.URLRecord, 0)

	for rows.Next() {
		var id, original, short, userID string

		err := rows.Scan(&id, &original, &short, &userID)
		if err != nil {
			r.logger.Error(fmt.Sprintf("FindByUserID error=%s", err.Error()))
			return nil, nil
		}

		res = append(res, storage.URLRecord{ID: id, Original: original, Short: short, UserID: userID})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &res, nil
}

func (r *URLRepository) PingContext(c context.Context) error {
	return r.db.PingContext(c)
}
