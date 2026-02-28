package controllers

import (
	"database/sql"
	"syncbeats-backend/db"
	"syncbeats-backend/models"
)

func getDB() *sql.DB {
	return db.GetInstance()
}

func AddFileToDb(fileId string, fileName string, path string) error {
	query := `INSERT INTO files (fileId, fileName, filePath) VALUES (?, ?, ?)`
	_, err := getDB().Exec(query, fileId, fileName, path)
	return err
}

func GetFilePathFromId(fileId string) (string, error) {
	query := `SELECT filePath FROM files WHERE fileId = ?`
	var filePath string
	err := getDB().QueryRow(query, fileId).Scan(&filePath)
	return filePath, err
}

func FileExists(fileId string) (bool, error) {
	query := `SELECT COUNT(*) FROM files WHERE fileId = ?`
	var count int
	err := getDB().QueryRow(query, fileId).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetAllFiles() ([]models.FileListItemModel, error) {
	query := `SELECT fileId, fileName FROM files`
	rows, err := getDB().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileListItemModel
	for rows.Next() {
		var item models.FileListItemModel
		if err := rows.Scan(&item.FileId, &item.FileName); err != nil {
			return nil, err
		}
		files = append(files, item)
	}
	return files, nil
}
