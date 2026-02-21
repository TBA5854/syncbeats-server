package utils

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func MustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("MustMarshal: " + err.Error())
	}
	return b
}

func BGCtx() context.Context {
	return context.Background()
}
