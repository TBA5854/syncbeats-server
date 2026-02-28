package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"

	"syncbeats-backend/controllers"
	"syncbeats-backend/db"
	"syncbeats-backend/hub"
	"syncbeats-backend/routes"
	"syncbeats-backend/services"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "syncbeats.db"
	}
	if err := db.Init(dbPath); err != nil {
		log.Fatalf("sqlite init: %v", err)
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	if err := db.InitRedis(redisAddr); err != nil {
		log.Fatalf("redis init: %v", err)
	}

	h := hub.New()
	roomSvc := services.NewRoomService(db.GetRedisInstance(), db.GetInstance())
	wsCtrl := &controllers.WSController{
		Hub:         h,
		RoomService: roomSvc,
	}
	roomCtrl := &controllers.RoomController{
		RoomService: roomSvc,
	}

	e := echo.New()

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	routes.RegisterFileRoutes(e)
	routes.RegisterWSRoutes(e, wsCtrl)
	routes.RegisterRoomRoutes(e, roomCtrl)

	srv := echo.StartConfig{Address: ":3000"}
	if err := srv.Start(context.Background(), e); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
