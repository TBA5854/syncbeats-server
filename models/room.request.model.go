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

// Queue operation payloads

type QueueAddPayload struct {
	RoomID    string `json:"room_id"`
	TrackHash string `json:"track_hash"`
	Position  int    `json:"position"` // -1 means append to end
}

type QueueRemovePayload struct {
	RoomID string `json:"room_id"`
	Index  int    `json:"index"`
}

type QueueMovePayload struct {
	RoomID string `json:"room_id"`
	From   int    `json:"from"`
	To     int    `json:"to"`
}

// Queue navigation — from_index is the caller's view of current_index;
// the server only advances if it matches (idempotent).
type QueueNextPayload struct {
	RoomID    string `json:"room_id"`
	FromIndex int    `json:"from_index"`
}

type QueuePrevPayload struct {
	RoomID    string `json:"room_id"`
	FromIndex int    `json:"from_index"`
}

type QueuePlayAtPayload struct {
	RoomID string `json:"room_id"`
	Index  int    `json:"index"`
}
