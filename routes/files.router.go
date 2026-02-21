package routes

import (
	"syncbeats-backend/controllers"

	"github.com/labstack/echo/v5"
)

func RegisterFileRoutes(e *echo.Echo) {
	e.POST("/files/upload", controllers.UploadFile)
	e.GET("/files/download", controllers.DownloadFile)
}
