package routes

import (
	"syncbeats-backend/controllers"

	"github.com/labstack/echo/v5"
)

func RegisterRoomRoutes(e *echo.Echo, rc *controllers.RoomController) {
	e.GET("/rooms/list", rc.ListRooms)
}
