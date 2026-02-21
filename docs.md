# SyncBeats API

## REST

Base URL: `http://localhost:3000`

---

### GET /

Health check.

**Response 200**
```json
{ "status": "ok" }
```

---

### POST /files/upload

Upload an audio file. Uploading the same file twice returns the same `file_id` without storing a duplicate.

**Request** — `multipart/form-data`

| Field | Type |
|---|---|
| `file` | file |

**Response 200**
```json
{ "file_id": "d41d8cd98f00b204e9800998ecf8427e" }
```

`file_id` is the MD5 hex digest of the file content. Use it as `track_hash` in `track:set`.

**Errors**

| Status | Body |
|---|---|
| 400 | `{ "error": "file not found" }` |
| 500 | `{ "error": "database error" }` |
| 500 | `{ "error": "failed to write file" }` |

---

### GET /files/download

Download a file by ID.

**Query params**

| Param | Type |
|---|---|
| `file_id` | string |

**Response 200** — raw file bytes

**Errors**

| Status | Body |
|---|---|
| 404 | `{ "error": "file not found" }` |

---

## WebSocket

**Endpoint:** `GET /ws?user_id=<id>`

`user_id` is required. All messages use the envelope format:

```json
{ "event": "<name>", "payload": { ... } }
```

On disconnect the server automatically removes the client from the hub and Redis member set.

---

### Error envelope

Sent to the triggering client only on any failure.

```json
{
  "event": "error",
  "payload": {
    "code": "JOIN_FAILED",
    "message": "room \"room-bob\" not found: redis: nil"
  }
}
```

| Code | Cause |
|---|---|
| `BAD_ENVELOPE` | Message is not valid JSON or missing fields |
| `BAD_PAYLOAD` | Payload fields are missing or wrong type |
| `UNKNOWN_EVENT` | Unrecognised event name |
| `CREATE_FAILED` | Redis write failed |
| `JOIN_FAILED` | Room does not exist or Redis error |
| `STATE_FETCH_FAILED` | Room does not exist or Redis error |
| `TRACK_SET_FAILED` | `track_hash` not in SQLite, or Redis error |
| `PLAY_FAILED` | No track set, or Redis error |
| `PAUSE_FAILED` | Redis error |
| `SEEK_FAILED` | Redis error |

---

### room:create

**Send**
```json
{
  "event": "room:create",
  "payload": {
    "user_id": "alice",
    "room_name": "chill vibes"
  }
}
```

**Receive — sender** `room:created`
```json
{
  "event": "room:created",
  "payload": {
    "room_id": "room-alice",
    "name": "chill vibes",
    "owner_id": "alice",
    "track_hash": "",
    "is_playing": false,
    "position": 0,
    "start_at": 0
  }
}
```

> `room_id` is derived as `room-{user_id}`.

---

### room:join

**Send**
```json
{
  "event": "room:join",
  "payload": {
    "user_id": "bob",
    "room_id": "room-alice"
  }
}
```

**Receive — sender** `room:joined`
```json
{
  "event": "room:joined",
  "payload": {
    "room_id": "room-alice",
    "name": "chill vibes",
    "owner_id": "alice",
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
    "is_playing": false,
    "position": 42.5,
    "start_at": 0
  }
}
```

**Broadcast — existing members** `room:member:joined`
```json
{
  "event": "room:member:joined",
  "payload": { "user_id": "bob" }
}
```

---

### room:leave

**Send**
```json
{
  "event": "room:leave",
  "payload": {
    "user_id": "bob",
    "room_id": "room-alice"
  }
}
```

**Receive — sender** `room:left`
```json
{
  "event": "room:left",
  "payload": { "room_id": "room-alice" }
}
```

**Broadcast — remaining members** `room:member:left`
```json
{
  "event": "room:member:left",
  "payload": { "user_id": "bob" }
}
```

---

### room:state:request

**Send**
```json
{
  "event": "room:state:request",
  "payload": { "room_id": "room-alice" }
}
```

**Receive — sender** `room:state`
```json
{
  "event": "room:state",
  "payload": {
    "room_id": "room-alice",
    "name": "chill vibes",
    "owner_id": "alice",
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
    "is_playing": true,
    "position": 42.5,
    "start_at": 1708512000300
  }
}
```

---

### track:set

Sets the current track. The `track_hash` must exist in the file library. Resets playback to paused at position 0.

**Send**
```json
{
  "event": "track:set",
  "payload": {
    "user_id": "alice",
    "room_id": "room-alice",
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e"
  }
}
```

**Broadcast — all members** `track:changed`
```json
{
  "event": "track:changed",
  "payload": {
    "room_id": "room-alice",
    "name": "chill vibes",
    "owner_id": "alice",
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
    "is_playing": false,
    "position": 0,
    "start_at": 0
  }
}
```

---

### sync:ntp

Clock sync exchange. Run before playback to calibrate client clock offset.

**Send**
```json
{
  "event": "sync:ntp",
  "payload": { "t1": 1708512000000 }
}
```

`t1` — client Unix milliseconds at send time.

**Receive — sender** `sync:ntp:pong`
```json
{
  "event": "sync:ntp:pong",
  "payload": {
    "t1": 1708512000000,
    "t2": 1708512000012,
    "t3": 1708512000013
  }
}
```

| Field | Description |
|---|---|
| `t1` | Echoed client send time |
| `t2` | Server receive time |
| `t3` | Server send time |

Client-side offset (T4 = client receive time):
```
RTT    = (T4 - T1) - (T3 - T2)
offset = ((T2 - T1) + (T3 - T4)) / 2
```

Run 3–5 times and take the median. Apply when scheduling playback: `scheduleAt = start_at + offset`.

---

### sync:play

Starts playback. Server sets `start_at = now + 300ms` and broadcasts to all members.

**Send**
```json
{
  "event": "sync:play",
  "payload": {
    "user_id": "alice",
    "room_id": "room-alice",
    "position": 42.5
  }
}
```

**Broadcast — all members** `sync:play`
```json
{
  "event": "sync:play",
  "payload": {
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
    "position": 42.5,
    "start_at": 1708512000300
  }
}
```

| Field | Description |
|---|---|
| `track_hash` | File to play |
| `position` | Seconds into the track to seek before playing |
| `start_at` | Unix ms — absolute moment all clients begin playback |

Client scheduling:
```
seek(position)
setTimeout(() => play(), (start_at + ntpOffset) - Date.now())
```

---

### sync:pause

**Send**
```json
{
  "event": "sync:pause",
  "payload": {
    "user_id": "alice",
    "room_id": "room-alice",
    "position": 61.3
  }
}
```

`position` — client's current playback position in seconds at pause time.

**Broadcast — all members** `sync:pause`
```json
{
  "event": "sync:pause",
  "payload": { "position": 61.3 }
}
```

---

### sync:seek

**Send**
```json
{
  "event": "sync:seek",
  "payload": {
    "user_id": "alice",
    "room_id": "room-alice",
    "position": 120.0
  }
}
```

**Broadcast — all members** `sync:seek`
```json
{
  "event": "sync:seek",
  "payload": {
    "position": 120.0,
    "start_at": 1708512005300
  }
}
```

`start_at` is non-zero only when the room was playing at seek time — treat it the same as `sync:play`. If `start_at` is `0` the room is paused; just seek to `position`.

---

## Typical session flow

```
Client A                          Server                          Client B
   |                                 |                                |
   |-- POST /files/upload ---------->|                                |
   |<-- { file_id: "abc..." } -------|                                |
   |                                 |                                |
   |-- WS /ws?user_id=alice -------->|                                |
   |-- room:create ----------------->|                                |
   |<-- room:created ----------------|                                |
   |                                 |                                |
   |                                 |<-- WS /ws?user_id=bob ---------|
   |                                 |<-- room:join ------------------|
   |<-- room:member:joined -----------|                                |
   |                                 |-- room:joined ---------------->|
   |                                 |                                |
   |-- sync:ntp (x3) --------------->|                                |
   |<-- sync:ntp:pong (x3) ----------|                                |
   |                                 |                                |
   |-- track:set ------------------->|                                |
   |<-- track:changed ---------------|-- track:changed -------------->|
   |                                 |                                |
   |-- sync:play ------------------->|                                |
   |<-- sync:play -------------------|-- sync:play ------------------>|
```
