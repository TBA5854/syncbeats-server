package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"

	"syncbeats-backend/hub"
	"syncbeats-backend/models"
	"syncbeats-backend/services"
	"syncbeats-backend/utils"
)

type WSController struct {
	Hub         *hub.Hub
	RoomService *services.RoomService
}

func (wc *WSController) HandleWS(c *echo.Context) error {
	userID := (*c).QueryParam("user_id")
	if userID == "" {
		return (*c).JSON(http.StatusBadRequest, map[string]string{"error": "user_id required"})
	}

	conn, err := utils.Upgrader.Upgrade((*c).Response(), (*c).Request(), nil)
	if err != nil {
		return err
	}

	client := &hub.Client{
		UserID: userID,
		Conn:   conn,
	}
	wc.Hub.Register(client)
	defer wc.Hub.Unregister(client)

	wc.readLoop(client)
	return nil
}

func (wc *WSController) readLoop(client *hub.Client) {
	for {
		_, raw, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var env models.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			wc.sendError(client, "BAD_ENVELOPE", "invalid JSON envelope")
			continue
		}

		wc.dispatch(client, env)
	}
}

func (wc *WSController) dispatch(client *hub.Client, env models.Envelope) {
	switch env.Event {
	case "room:create":
		wc.handleRoomCreate(client, env.Payload)
	case "room:join":
		wc.handleRoomJoin(client, env.Payload)
	case "room:leave":
		wc.handleRoomLeave(client, env.Payload)
	case "room:state:request":
		wc.handleRoomStateRequest(client, env.Payload)
	case "track:set":
		wc.handleTrackSet(client, env.Payload)
	case "sync:ntp":
		wc.handleNTP(client, env.Payload)
	case "sync:play":
		wc.handleSyncPlay(client, env.Payload)
	case "sync:pause":
		wc.handleSyncPause(client, env.Payload)
	case "sync:seek":
		wc.handleSyncSeek(client, env.Payload)
	default:
		wc.sendError(client, "UNKNOWN_EVENT", "unknown event: "+env.Event)
	}
}

func (wc *WSController) handleRoomCreate(client *hub.Client, raw json.RawMessage) {
	var p models.RoomCreatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.CreateRoom(utils.BGCtx(), p.UserID, p.RoomName)
	if err != nil {
		wc.sendError(client, "CREATE_FAILED", err.Error())
		return
	}

	wc.Hub.AddToRoom(client, state.RoomID)

	_ = client.Send(models.Envelope{
		Event:   "room:created",
		Payload: utils.MustMarshal(state),
	})
}

func (wc *WSController) handleRoomJoin(client *hub.Client, raw json.RawMessage) {
	var p models.RoomJoinPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.JoinRoom(utils.BGCtx(), p.RoomID, p.UserID)
	if err != nil {
		wc.sendError(client, "JOIN_FAILED", err.Error())
		return
	}

	wc.Hub.AddToRoom(client, p.RoomID)

	_ = client.Send(models.Envelope{
		Event:   "room:joined",
		Payload: utils.MustMarshal(state),
	})

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event:   "room:member:joined",
		Payload: utils.MustMarshal(map[string]string{"user_id": p.UserID}),
	}, client)
}

func (wc *WSController) handleRoomLeave(client *hub.Client, raw json.RawMessage) {
	var p models.RoomLeavePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	if err := wc.RoomService.LeaveRoom(utils.BGCtx(), p.RoomID, p.UserID); err != nil {
		log.Printf("LeaveRoom redis error: %v", err)
	}

	wc.Hub.RemoveFromRoom(client, p.RoomID)

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event:   "room:member:left",
		Payload: utils.MustMarshal(map[string]string{"user_id": p.UserID}),
	})

	_ = client.Send(models.Envelope{
		Event:   "room:left",
		Payload: utils.MustMarshal(map[string]string{"room_id": p.RoomID}),
	})
}

func (wc *WSController) handleRoomStateRequest(client *hub.Client, raw json.RawMessage) {
	var p models.RoomStateRequestPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.GetRoomState(utils.BGCtx(), p.RoomID)
	if err != nil {
		wc.sendError(client, "STATE_FETCH_FAILED", err.Error())
		return
	}

	_ = client.Send(models.Envelope{
		Event:   "room:state",
		Payload: utils.MustMarshal(state),
	})
}

func (wc *WSController) handleTrackSet(client *hub.Client, raw json.RawMessage) {
	var p models.TrackSetPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.SetTrack(utils.BGCtx(), p.RoomID, p.TrackHash)
	if err != nil {
		wc.sendError(client, "TRACK_SET_FAILED", err.Error())
		return
	}

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event:   "track:changed",
		Payload: utils.MustMarshal(state),
	})
}

// Client computes (T4 = client receive time):
//
//	RTT    = (T4 - T1) - (T3 - T2)
//	offset = ((T2 - T1) + (T3 - T4)) / 2
func (wc *WSController) handleNTP(client *hub.Client, raw json.RawMessage) {
	var p models.NTPPingPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	result := services.HandlePing(p.T1)
	result.Seal()

	_ = client.Send(models.Envelope{
		Event: "sync:ntp:pong",
		Payload: utils.MustMarshal(models.NTPPongPayload{
			T1: result.T1,
			T2: result.T2,
			T3: result.T3,
		}),
	})
}

func (wc *WSController) handleSyncPlay(client *hub.Client, raw json.RawMessage) {
	var p models.SyncPlayPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.Play(utils.BGCtx(), p.RoomID, p.Position)
	if err != nil {
		wc.sendError(client, "PLAY_FAILED", err.Error())
		return
	}

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event: "sync:play",
		Payload: utils.MustMarshal(models.SyncPlayBroadcast{
			TrackHash: state.TrackHash,
			Position:  state.Position,
			StartAt:   state.StartAt,
		}),
	})
}

func (wc *WSController) handleSyncPause(client *hub.Client, raw json.RawMessage) {
	var p models.SyncPausePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.Pause(utils.BGCtx(), p.RoomID, p.Position)
	if err != nil {
		wc.sendError(client, "PAUSE_FAILED", err.Error())
		return
	}

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event: "sync:pause",
		Payload: utils.MustMarshal(models.SyncPauseBroadcast{
			Position: state.Position,
		}),
	})
}

func (wc *WSController) handleSyncSeek(client *hub.Client, raw json.RawMessage) {
	var p models.SyncSeekPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		wc.sendError(client, "BAD_PAYLOAD", err.Error())
		return
	}

	state, err := wc.RoomService.Seek(utils.BGCtx(), p.RoomID, p.Position)
	if err != nil {
		wc.sendError(client, "SEEK_FAILED", err.Error())
		return
	}

	wc.Hub.BroadcastToRoom(p.RoomID, models.Envelope{
		Event: "sync:seek",
		Payload: utils.MustMarshal(models.SyncSeekBroadcast{
			Position: state.Position,
			StartAt:  state.StartAt,
		}),
	})
}

func (wc *WSController) sendError(client *hub.Client, code, message string) {
	_ = client.Send(models.Envelope{
		Event:   "error",
		Payload: utils.MustMarshal(models.ErrorPayload{Code: code, Message: message}),
	})
}
