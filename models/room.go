package models

type RoomState struct {
	RoomID    string  `json:"room_id"`
	Name      string  `json:"name"`
	OwnerID   string  `json:"owner_id"`
	TrackHash string  `json:"track_hash"`
	IsPlaying bool    `json:"is_playing"`
	Position  float64 `json:"position"`
	StartAt   int64   `json:"start_at"`
}
