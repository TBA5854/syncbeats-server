package controllers

import (
	"context"
	"fmt"
	"syncbeats-backend/db"
	"syncbeats-backend/models"
)

func CreateRoom(roomName, userID string) (*models.CreateRoomResponse, error) {
	ctx := context.Background()
	redis := db.GetRedisInstance()

	roomID := fmt.Sprintf("room:%s", userID) 

	err := redis.HSet(ctx, roomID, map[string]any{
		"name":  roomName,
		"owner": userID,
	}).Err()
	if err != nil {
		return nil, err
	}

	return &models.CreateRoomResponse{
		RoomID: roomID,
		Name:   roomName,
	}, nil
}

func JoinRoom(roomID, userID string) (*models.JoinRoomResponse, error) {
	ctx := context.Background()
	redis := db.GetRedisInstance()

	err := redis.SAdd(ctx, fmt.Sprintf("%s:users", roomID), userID).Err()
	if err != nil {
		return nil, err
	}

	return &models.JoinRoomResponse{
		RoomID: roomID,
		UserID: userID,
	}, nil
}
