package models

type CreateRoomRequest struct {
	RoomName string `json:"room_name"`
	UserID   string `json:"user_id"`
}

type JoinRoomRequest struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
}

type RoomCreatePayload struct {
	UserID   string `json:"user_id"`
	RoomName string `json:"room_name"`
}

type RoomJoinPayload struct {
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`
}

type RoomLeavePayload struct {
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`
}

type RoomStateRequestPayload struct {
	RoomID string `json:"room_id"`
}

type TrackSetPayload struct {
	UserID    string `json:"user_id"`
	RoomID    string `json:"room_id"`
	TrackHash string `json:"track_hash"`
}

type NTPPingPayload struct {
	T1 int64 `json:"t1"`
}

type SyncPlayPayload struct {
	UserID   string  `json:"user_id"`
	RoomID   string  `json:"room_id"`
	Position float64 `json:"position"`
}

type SyncPausePayload struct {
	UserID   string  `json:"user_id"`
	RoomID   string  `json:"room_id"`
	Position float64 `json:"position"`
}

type SyncSeekPayload struct {
	UserID   string  `json:"user_id"`
	RoomID   string  `json:"room_id"`
	Position float64 `json:"position"`
}
