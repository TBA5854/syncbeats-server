package routes

import (
	"syncbeats-backend/controllers"

	"github.com/labstack/echo/v5"
)

func RegisterWSRoutes(e *echo.Echo, wc *controllers.WSController) {
	e.GET("/ws", wc.HandleWS)
}
