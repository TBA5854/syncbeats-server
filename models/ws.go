package models

import "encoding/json"

type Envelope struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}
