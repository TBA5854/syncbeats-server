package controllers

import (
	"net/http"
	"syncbeats-backend/services"

	"github.com/labstack/echo/v5"
)

type RoomController struct {
	RoomService *services.RoomService
}

func (rc *RoomController) ListRooms(c *echo.Context) error {
	rooms, err := rc.RoomService.ListRooms((*c).Request().Context())
	if err != nil {
		return (*c).JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return (*c).JSON(http.StatusOK, rooms)
}
