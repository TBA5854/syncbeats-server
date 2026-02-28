package db

import (
	"database/sql"
	"log"
	"sync"

	_ "modernc.org/sqlite"
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
		DB, err = sql.Open("sqlite", dbPath)
		if err != nil {
			log.Fatal(err)
		}

		err = DB.Ping()
		if err != nil {
			log.Fatal(err)
		}

		createTableSQL := `CREATE TABLE IF NOT EXISTS files (
			fileId TEXT PRIMARY KEY,
			fileName TEXT NOT NULL,
			filePath TEXT NOT NULL
		);`

		_, err = DB.Exec(createTableSQL)
		if err != nil {
			log.Fatal(err)
		}
	})
	return err
}
