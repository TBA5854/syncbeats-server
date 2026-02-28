package hub

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID   string
	Username string
	RoomID   string
	Conn     *websocket.Conn
	mu       sync.Mutex
}

func (c *Client) Send(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	rooms   map[string]map[*Client]struct{}
}

func New() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		rooms:   make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	if c.RoomID != "" {
		h.removeFromRoom(c, c.RoomID)
	}
}

func (h *Hub) AddToRoom(c *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.RoomID != "" && c.RoomID != roomID {
		h.removeFromRoom(c, c.RoomID)
	}
	c.RoomID = roomID
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]struct{})
	}
	h.rooms[roomID][c] = struct{}{}
}

func (h *Hub) RemoveFromRoom(c *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeFromRoom(c, roomID)
}

func (h *Hub) removeFromRoom(c *Client, roomID string) {
	if members, ok := h.rooms[roomID]; ok {
		delete(members, c)
		if len(members) == 0 {
			delete(h.rooms, roomID)
		}
	}
	if c.RoomID == roomID {
		c.RoomID = ""
	}
}

func (h *Hub) BroadcastToRoom(roomID string, v any, exclude ...*Client) {
	h.mu.RLock()
	members := h.rooms[roomID]
	targets := make([]*Client, 0, len(members))
	for c := range members {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	excSet := make(map[*Client]struct{}, len(exclude))
	for _, c := range exclude {
		excSet[c] = struct{}{}
	}

	for _, c := range targets {
		if _, skip := excSet[c]; skip {
			continue
		}
		_ = c.Send(v)
	}
}

func (h *Hub) SendToClient(userID string, v any) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.UserID == userID {
			_ = c.Send(v)
			return true
		}
	}
	return false
}
