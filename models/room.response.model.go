package models

type CreateRoomResponse struct {
	RoomID string `json:"room_id"`
	Name   string `json:"name"`
}

type JoinRoomResponse struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}
