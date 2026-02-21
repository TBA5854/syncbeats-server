package controllers

import (
	"syncbeats-backend/db"
)

var db_instance = db.GetInstance()

func AddFileToDb(fileId string, path string) error {
	query := `INSERT INTO files (fileId, filePath) VALUES (?, ?)`
	_, err := db_instance.Exec(query, fileId, path)
	return err
}

func GetFilePathFromId(fileId string) (string, error) {
	query := `SELECT filePath FROM files WHERE fileId = ?`
	var filePath string
	err := db_instance.QueryRow(query, fileId).Scan(&filePath)
	return filePath, err
}

func FileExists(fileId string) (bool, error) {
	query := `SELECT COUNT(*) FROM files WHERE fileId = ?`
	var count int
	err := db_instance.QueryRow(query, fileId).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
