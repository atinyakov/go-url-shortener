package repository

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDb(host string) *sql.DB {
	ps := fmt.Sprintf("host=%s user=%s port=5432 password=%s dbname=%s sslmode=disable",
		host, `db_tus`, `qwerty`, `urls`)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		panic(err)
	}

	fmt.Println("Database connected and table ready.")
	return db
}
