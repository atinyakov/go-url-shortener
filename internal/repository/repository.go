package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/atinyakov/go-url-shortener/internal/storage"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

var ErrConflict = errors.New("data conflict")

func InitDB(ps string) *sql.DB {
	db, err := sql.Open("pgx", ps)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	// Create table if not exists
	createTable := `
		CREATE TABLE IF NOT EXISTS url_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		original_url TEXT UNIQUE NOT NULL,
		short_url TEXT UNIQUE NOT NULL
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
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

func (r *URLRepository) Write(v storage.URLRecord) (*storage.URLRecord, error) {
	var existing = v

	err := r.db.QueryRow(
		`INSERT INTO url_records(original_url, short_url) 
		 VALUES ($1, $2) 
		 ON CONFLICT (original_url) DO NOTHING 
		 RETURNING original_url, short_url, id;`,
		v.Original, v.Short,
	).Scan(&existing.Original, &existing.Short, &existing.ID)

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

func (r *URLRepository) WriteAll(rs []storage.URLRecord) error {
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

	for _, v := range rs {
		stmt, err := tx.Prepare(`
			INSERT INTO url_records(original_url, short_url, id) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (original_url) DO NOTHING 
			RETURNING original_url, short_url, id;
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(v.Original, v.Short, v.ID)

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

func (r *URLRepository) Read() ([]storage.URLRecord, error) {
	rows, err := r.db.Query("SELECT * FROM url_records;")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	records := make([]storage.URLRecord, 0)

	for rows.Next() {
		var r storage.URLRecord
		err = rows.Scan(&r.ID, &r.Original, &r.Short)
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

func (r *URLRepository) FindByShort(s string) (*storage.URLRecord, error) {
	row := r.db.QueryRow("SELECT * FROM url_records WHERE short_url = $1;", s)

	var id, originalURL, shortURL string

	err := row.Scan(&id, &originalURL, &shortURL)
	if err != nil {
		r.logger.Error("FindByShort err=", zap.String("error", err.Error()))
		return nil, err
	}

	return &storage.URLRecord{
		ID:       id,
		Original: originalURL,
		Short:    shortURL,
	}, nil
}

func (r *URLRepository) FindByLong(long string) (*storage.URLRecord, error) {
	row := r.db.QueryRow("SELECT * FROM url_records WHERE short_url = $1;", long)

	var id, originalURL, shortURL string

	err := row.Scan(&id, &originalURL, &shortURL)
	if err != nil {
		r.logger.Error("FindByShort err=", zap.String("error", err.Error()))
		return nil, err
	}

	return &storage.URLRecord{
		ID:       id,
		Original: originalURL,
		Short:    shortURL,
	}, nil
}

func (r *URLRepository) FindByID(s string) (storage.URLRecord, error) {
	row := r.db.QueryRow("SELECT * FROM url_records WHERE id = $1;", s)

	var id, original, short string

	err := row.Scan(&id, &original, &short)
	if err != nil {
		r.logger.Error("FindByID error=", zap.String("error", err.Error()))
		return storage.URLRecord{}, nil
	}

	return storage.URLRecord{
		ID:       id,
		Original: original,
		Short:    short,
	}, nil
}

func (r *URLRepository) PingContext(c context.Context) error {
	return r.db.PingContext(c)
}
