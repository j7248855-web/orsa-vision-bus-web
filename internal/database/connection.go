package database

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Подключение к базе данных
func Connection() *sqlx.DB {
	conn := fmt.Sprintf("user=%v dbname=%v password=%v host=%v port=%v sslmode=disable", os.Getenv("DB_USER"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"))
	db, err := sqlx.Connect("postgres", conn)
	if err != nil {
		log.Println("Не удалось подключиться к базе:", err)
		return nil
	}
	return db
}
