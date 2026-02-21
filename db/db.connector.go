package db

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB   *sql.DB
	once sync.Once
)

func GetInstance() *sql.DB {
	return DB
}

func Init(dbPath string) error {
	var err error
	once.Do(func() {
		DB, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatal(err)
		}

		err = DB.Ping()
		if err != nil {
			log.Fatal(err)
		}

		createTableSQL := `CREATE TABLE IF NOT EXISTS files (
			fileId TEXT PRIMARY KEY,
			filePath TEXT NOT NULL
		);`

		_, err = DB.Exec(createTableSQL)
		if err != nil {
			log.Fatal(err)
		}
	})
	return err
}
