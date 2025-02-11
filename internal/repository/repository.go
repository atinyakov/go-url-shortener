package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/atinyakov/go-url-shortener/internal/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

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
		original_url TEXT NOT NULL,
		short_url TEXT UNIQUE NOT NULL
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connected and table ready.")
	return db
}

type URLRepository struct {
	db *sql.DB
}

func CreateURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		db: db,
	}
}

func (r *URLRepository) Write(v storage.URLRecord) error {
	_, err := r.db.Exec("INSERT INTO url_records(original_url , short_url) VALUES ($1, $2)", v.Original, v.Short)

	if err != nil {
		return err
	}

	return nil
}

func (r *URLRepository) WriteAll(rs []storage.URLRecord) ([]storage.URLRecord, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	result := make([]storage.URLRecord, 0)

	for _, v := range rs {
		var id string
		err = tx.QueryRow("INSERT INTO url_records(original_url , short_url) VALUES ($1, $2) RETURNING id", v.Original, v.Short).Scan(&id)
		if err != nil {
			tx.Rollback()
			fmt.Println("ROOOOOOOOOOOOOOOOOLBACK!", err.Error())
			return nil, err
		}

		result = append(result, storage.URLRecord{ID: id, Original: v.Original, Short: v.Short})

	}

	tx.Commit()

	return result, nil
}

func (r *URLRepository) Read() ([]storage.URLRecord, error) {
	rows, err := r.db.Query("SELECT * FROM url_records")

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

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return records, nil

}

func (r *URLRepository) FindByShort(s string) (storage.URLRecord, error) {
	row := r.db.QueryRow("SELECT * FROM url_records WHERE short_url = $1", s)

	var id, originalURL, shortURL string
	fmt.Println("res")

	err := row.Scan(&id, &originalURL, &shortURL)
	if err != nil {
		fmt.Println(err.Error())
		return storage.URLRecord{}, nil
	}

	res := storage.URLRecord{
		ID:       id,
		Original: originalURL,
		Short:    shortURL,
	}

	return res, nil

}

func (r *URLRepository) FindByOriginal(s string) (storage.URLRecord, error) {
	fmt.Println("repo got long", s)
	row := r.db.QueryRow("SELECT * FROM url_records WHERE original_url = $1;", s)

	var id, original, short string

	err := row.Scan(&id, &original, &short)
	if err != nil {
		fmt.Println(err.Error())
		return storage.URLRecord{}, nil
	}

	return storage.URLRecord{
		ID:       id,
		Original: original,
		Short:    short,
	}, nil

}

func (r *URLRepository) FindByID(s string) (storage.URLRecord, error) {
	row := r.db.QueryRow("SELECT * FROM url_records WHERE id = $1;", s)

	var id, original, short string

	err := row.Scan(&id, &original, &short)
	if err != nil {
		fmt.Println("FindByID:", err.Error())
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
