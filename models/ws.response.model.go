package models

type NTPPongPayload struct {
	T1 int64 `json:"t1"`
	T2 int64 `json:"t2"`
	T3 int64 `json:"t3"`
}

type SyncPlayBroadcast struct {
	TrackHash string  `json:"track_hash"`
	Position  float64 `json:"position"`
	StartAt   int64   `json:"start_at"`
}

type SyncPauseBroadcast struct {
	Position float64 `json:"position"`
}

type SyncSeekBroadcast struct {
	Position float64 `json:"position"`
	StartAt  int64   `json:"start_at"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
