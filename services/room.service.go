package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"syncbeats-backend/models"

	"github.com/redis/go-redis/v9"
)

const playDelay = 300 * time.Millisecond

func roomKey(roomID string) string        { return "room:" + roomID }
func roomMembersKey(roomID string) string { return "room:" + roomID + ":members" }

type RoomService struct {
	rdb *redis.Client
	db  *sql.DB
}

func NewRoomService(rdb *redis.Client, db *sql.DB) *RoomService {
	return &RoomService{rdb: rdb, db: db}
}

func (s *RoomService) CreateRoom(ctx context.Context, ownerID, roomName string) (*models.RoomState, error) {
	roomID := fmt.Sprintf("room-%s", ownerID)
	key := roomKey(roomID)

	state := &models.RoomState{
		RoomID:  roomID,
		Name:    roomName,
		OwnerID: ownerID,
	}

	fields := map[string]any{
		"name":          state.Name,
		"owner_id":      state.OwnerID,
		"track_hash":    "",
		"is_playing":    "false",
		"position":      "0",
		"start_at":      "0",
		"current_index": "-1",
	}
	if err := s.rdb.HSet(ctx, key, fields).Err(); err != nil {
		return nil, fmt.Errorf("CreateRoom hset: %w", err)
	}

	// Track the room in a global set
	if err := s.rdb.SAdd(ctx, "rooms:all", roomID).Err(); err != nil {
		log.Printf("CreateRoom sadd rooms:all error: %v", err)
	}

	return state, nil
}

func (s *RoomService) JoinRoom(ctx context.Context, roomID, userID string) (*models.RoomState, error) {
	state, err := s.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if err := s.rdb.SAdd(ctx, roomMembersKey(roomID), userID).Err(); err != nil {
		return nil, fmt.Errorf("JoinRoom sadd: %w", err)
	}

	return state, nil
}

func (s *RoomService) LeaveRoom(ctx context.Context, roomID, userID string) (bool, error) {
	if err := s.rdb.SRem(ctx, roomMembersKey(roomID), userID).Err(); err != nil {
		return false, fmt.Errorf("LeaveRoom srem: %w", err)
	}

	// Check if room is empty
	count, err := s.rdb.SCard(ctx, roomMembersKey(roomID)).Result()
	if err != nil {
		return false, fmt.Errorf("LeaveRoom scard: %w", err)
	}

	if count == 0 {
		// Delete the room
		_ = s.rdb.Del(ctx, roomKey(roomID)).Err()
		_ = s.rdb.Del(ctx, roomMembersKey(roomID)).Err()
		_ = s.rdb.SRem(ctx, "rooms:all", roomID).Err()
		return true, nil
	}

	return false, nil
}

func (s *RoomService) GetRoomState(ctx context.Context, roomID string) (*models.RoomState, error) {
	vals, err := s.rdb.HGetAll(ctx, roomKey(roomID)).Result()
	if err != nil {
		return nil, fmt.Errorf("GetRoomState hgetall: %w", err)
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("room %q not found: %w", roomID, redis.Nil)
	}
	return parseRoomState(roomID, vals)
}

func (s *RoomService) ListRooms(ctx context.Context) ([]*models.RoomState, error) {
	roomIDs, err := s.rdb.SMembers(ctx, "rooms:all").Result()
	if err != nil {
		return nil, fmt.Errorf("ListRooms smembers: %w", err)
	}

	rooms := make([]*models.RoomState, 0, len(roomIDs))
	for _, id := range roomIDs {
		state, err := s.GetRoomState(ctx, id)
		if err != nil {
			log.Printf("ListRooms: error fetching state for %s: %v", id, err)
			continue
		}
		rooms = append(rooms, state)
	}

	return rooms, nil
}

func (s *RoomService) SetTrack(ctx context.Context, roomID, trackHash string) (*models.RoomState, error) {
	var filePath string
	err := s.db.QueryRowContext(ctx, `SELECT filePath FROM files WHERE fileId = ?`, trackHash).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("track %q not found in library", trackHash)
	}
	if err != nil {
		return nil, fmt.Errorf("SetTrack db query: %w", err)
	}

	// Find the track's position in the queue (first occurrence), or -1.
	q, _ := s.getQueue(ctx, roomID)
	currentIndex := -1
	for i, h := range q {
		if h == trackHash {
			currentIndex = i
			break
		}
	}

	fields := map[string]any{
		"track_hash":    trackHash,
		"is_playing":    "false",
		"position":      "0",
		"start_at":      "0",
		"current_index": fmt.Sprintf("%d", currentIndex),
	}
	if err := s.rdb.HSet(ctx, roomKey(roomID), fields).Err(); err != nil {
		return nil, fmt.Errorf("SetTrack hset: %w", err)
	}

	return s.GetRoomState(ctx, roomID)
}

func (s *RoomService) Play(ctx context.Context, roomID string, position float64) (*models.RoomState, error) {
	state, err := s.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if state.TrackHash == "" {
		return nil, errors.New("no track set for this room")
	}

	startAt := time.Now().Add(playDelay).UnixMilli()

	fields := map[string]any{
		"is_playing": "true",
		"position":   fmt.Sprintf("%f", position),
		"start_at":   fmt.Sprintf("%d", startAt),
	}
	if err := s.rdb.HSet(ctx, roomKey(roomID), fields).Err(); err != nil {
		return nil, fmt.Errorf("Play hset: %w", err)
	}

	state.IsPlaying = true
	state.Position = position
	state.StartAt = startAt
	return state, nil
}

func (s *RoomService) Pause(ctx context.Context, roomID string, position float64) (*models.RoomState, error) {
	fields := map[string]any{
		"is_playing": "false",
		"position":   fmt.Sprintf("%f", position),
		"start_at":   "0",
	}
	if err := s.rdb.HSet(ctx, roomKey(roomID), fields).Err(); err != nil {
		return nil, fmt.Errorf("Pause hset: %w", err)
	}
	return s.GetRoomState(ctx, roomID)
}

func (s *RoomService) Seek(ctx context.Context, roomID string, position float64) (*models.RoomState, error) {
	state, err := s.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, err
	}

	var startAt int64
	if state.IsPlaying {
		startAt = time.Now().Add(playDelay).UnixMilli()
	}

	fields := map[string]any{
		"position": fmt.Sprintf("%f", position),
		"start_at": fmt.Sprintf("%d", startAt),
	}
	if err := s.rdb.HSet(ctx, roomKey(roomID), fields).Err(); err != nil {
		return nil, fmt.Errorf("Seek hset: %w", err)
	}

	state.Position = position
	state.StartAt = startAt
	return state, nil
}

func (s *RoomService) getQueue(ctx context.Context, roomID string) ([]string, error) {
	v, err := s.rdb.HGet(ctx, roomKey(roomID), "queue").Result()
	if err == redis.Nil {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var q []string
	if err := json.Unmarshal([]byte(v), &q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *RoomService) saveQueue(ctx context.Context, roomID string, q []string) error {
	b, err := json.Marshal(q)
	if err != nil {
		return err
	}
	return s.rdb.HSet(ctx, roomKey(roomID), "queue", string(b)).Err()
}

func (s *RoomService) QueueAdd(ctx context.Context, roomID, trackHash string, position int) ([]string, error) {
	q, err := s.getQueue(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("QueueAdd: %w", err)
	}

	if position < 0 || position >= len(q) {
		q = append(q, trackHash)
	} else {
		q = append(q[:position], append([]string{trackHash}, q[position:]...)...)
	}

	if err := s.saveQueue(ctx, roomID, q); err != nil {
		return nil, fmt.Errorf("QueueAdd: %w", err)
	}
	return q, nil
}

func (s *RoomService) QueueRemove(ctx context.Context, roomID string, index int) ([]string, error) {
	q, err := s.getQueue(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("QueueRemove: %w", err)
	}
	if index < 0 || index >= len(q) {
		return nil, fmt.Errorf("QueueRemove: index %d out of range", index)
	}

	q = append(q[:index], q[index+1:]...)

	if err := s.saveQueue(ctx, roomID, q); err != nil {
		return nil, fmt.Errorf("QueueRemove: %w", err)
	}
	return q, nil
}

func (s *RoomService) QueueMove(ctx context.Context, roomID string, from, to int) ([]string, error) {
	q, err := s.getQueue(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("QueueMove: %w", err)
	}
	if from < 0 || from >= len(q) || to < 0 || to >= len(q) {
		return nil, fmt.Errorf("QueueMove: index out of range")
	}

	item := q[from]
	q = append(q[:from], q[from+1:]...)
	q = append(q[:to], append([]string{item}, q[to:]...)...)

	if err := s.saveQueue(ctx, roomID, q); err != nil {
		return nil, fmt.Errorf("QueueMove: %w", err)
	}
	return q, nil
}

func parseRoomState(roomID string, vals map[string]string) (*models.RoomState, error) {
	state := &models.RoomState{
		RoomID:       roomID,
		Name:         vals["name"],
		OwnerID:      vals["owner_id"],
		TrackHash:    vals["track_hash"],
		CurrentIndex: -1,
	}

	state.IsPlaying = vals["is_playing"] == "true"

	if v := vals["position"]; v != "" {
		if err := json.Unmarshal([]byte(v), &state.Position); err != nil {
			fmt.Sscanf(v, "%f", &state.Position)
		}
	}

	if v := vals["start_at"]; v != "" {
		fmt.Sscanf(v, "%d", &state.StartAt)
	}

	if v := vals["current_index"]; v != "" {
		fmt.Sscanf(v, "%d", &state.CurrentIndex)
	}

	state.Queue = []string{}
	if v := vals["queue"]; v != "" {
		_ = json.Unmarshal([]byte(v), &state.Queue)
	}

	return state, nil
}

// queuePlayAt sets the track at index i as the currently playing track.
func (s *RoomService) queuePlayAt(ctx context.Context, roomID string, index int) (*models.RoomState, error) {
	q, err := s.getQueue(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(q) {
		return nil, fmt.Errorf("index %d out of range", index)
	}

	trackHash := q[index]

	// Validate track exists in library.
	var filePath string
	err = s.db.QueryRowContext(ctx, `SELECT filePath FROM files WHERE fileId = ?`, trackHash).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("track %q not found in library", trackHash)
	}
	if err != nil {
		return nil, fmt.Errorf("queuePlayAt db: %w", err)
	}

	fields := map[string]any{
		"track_hash":    trackHash,
		"is_playing":    "false",
		"position":      "0",
		"start_at":      "0",
		"current_index": fmt.Sprintf("%d", index),
	}
	if err := s.rdb.HSet(ctx, roomKey(roomID), fields).Err(); err != nil {
		return nil, fmt.Errorf("queuePlayAt hset: %w", err)
	}
	return s.GetRoomState(ctx, roomID)
}

// QueuePlayAt is the public version — plays by explicit index.
func (s *RoomService) QueuePlayAt(ctx context.Context, roomID string, index int) (*models.RoomState, error) {
	return s.queuePlayAt(ctx, roomID, index)
}

// QueueNext advances to the next track. Idempotent: only moves if stored
// current_index matches fromIndex, so concurrent calls are safe.
func (s *RoomService) QueueNext(ctx context.Context, roomID string, fromIndex int) (*models.RoomState, error) {
	state, err := s.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if state.CurrentIndex != fromIndex {
		return state, nil // already moved — return current state, no-op
	}
	return s.queuePlayAt(ctx, roomID, state.CurrentIndex+1)
}

// QueuePrev goes back one track. Same idempotency guarantee as QueueNext.
func (s *RoomService) QueuePrev(ctx context.Context, roomID string, fromIndex int) (*models.RoomState, error) {
	state, err := s.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if state.CurrentIndex != fromIndex {
		return state, nil
	}
	return s.queuePlayAt(ctx, roomID, state.CurrentIndex-1)
}

func (rs *RoomService) SetCurrentIndex(ctx context.Context, roomID string, index int) error {
	key := "room:" + roomID
	return rs.rdb.HSet(ctx, key, "currentIndex", index).Err()
}
