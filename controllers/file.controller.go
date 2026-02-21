package controllers

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syncbeats-backend/models"

	"github.com/labstack/echo/v5"
)

func UploadFile(c *echo.Context) error {
	req := new(models.FileUploadRequestModel)
	if err := (*c).Bind(req); err != nil {
		return (*c).JSON(400, map[string]string{"error": "invalid request"})
	}

	file, err := (*c).FormFile("file")
	if err != nil {
		return (*c).JSON(400, map[string]string{"error": "file not found"})
	}

	src, err := file.Open()
	if err != nil {
		return (*c).JSON(500, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, src); err != nil {
		return (*c).JSON(500, map[string]string{"error": "failed to compute hash"})
	}
	fileId := fmt.Sprintf("%x", hash.Sum(nil))

	exists, err := FileExists(fileId)
	if err != nil {
		return (*c).JSON(500, map[string]string{"error": "database error"})
	}

	if !exists {
		uploadDir := "uploads"
		os.MkdirAll(uploadDir, 0755)
		filePath := filepath.Join(uploadDir, fileId)

		src.Seek(0, 0)
		dst, err := os.Create(filePath)
		if err != nil {
			return (*c).JSON(500, map[string]string{"error": "failed to create file"})
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return (*c).JSON(500, map[string]string{"error": "failed to write file"})
		}

		if err := AddFileToDb(fileId, filePath); err != nil {
			return (*c).JSON(500, map[string]string{"error": "failed to add file to database"})
		}
	}

	return (*c).JSON(200, models.FileUploadResponseModel{FileId: fileId})
}

func DownloadFile(c *echo.Context) error {
	req := new(models.FileDownloadRequestModel)
	if err := (*c).Bind(req); err != nil {
		return (*c).JSON(400, map[string]string{"error": "invalid request"})
	}

	filePath, err := GetFilePathFromId(req.FileId)
	if err != nil {
		return (*c).JSON(404, map[string]string{"error": "file not found"})
	}

	return (*c).File(filePath)
}
