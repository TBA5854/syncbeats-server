package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"syncbeats-backend/models"
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
		"name":       state.Name,
		"owner_id":   state.OwnerID,
		"track_hash": "",
		"is_playing": "false",
		"position":   "0",
		"start_at":   "0",
	}
	if err := s.rdb.HSet(ctx, key, fields).Err(); err != nil {
		return nil, fmt.Errorf("CreateRoom hset: %w", err)
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

func (s *RoomService) LeaveRoom(ctx context.Context, roomID, userID string) error {
	return s.rdb.SRem(ctx, roomMembersKey(roomID), userID).Err()
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

func (s *RoomService) SetTrack(ctx context.Context, roomID, trackHash string) (*models.RoomState, error) {
	var filePath string
	err := s.db.QueryRowContext(ctx, `SELECT filePath FROM files WHERE fileId = ?`, trackHash).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("track %q not found in library", trackHash)
	}
	if err != nil {
		return nil, fmt.Errorf("SetTrack db query: %w", err)
	}

	fields := map[string]any{
		"track_hash": trackHash,
		"is_playing": "false",
		"position":   "0",
		"start_at":   "0",
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

func parseRoomState(roomID string, vals map[string]string) (*models.RoomState, error) {
	state := &models.RoomState{
		RoomID:    roomID,
		Name:      vals["name"],
		OwnerID:   vals["owner_id"],
		TrackHash: vals["track_hash"],
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

	return state, nil
}
