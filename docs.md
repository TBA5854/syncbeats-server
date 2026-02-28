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

| Field | Type | Notes |
|---|---|---|
| `file` | file | Required |
| `file_name` | string | Optional display name |
| `hash` | string | **Required.** MD5 hex of the file. Server verifies integrity and rejects on mismatch. |

**Response 200**
```json
{ "file_id": "d41d8cd98f00b204e9800998ecf8427e" }
```

`file_id` is the MD5 hex digest of the file content. Use it as `track_hash` in `track:set`.

**Errors**

| Status | Body | Cause |
|---|---|---|
| 400 | `{ "error": "hash_required" }` | `hash` field was not sent |
| 400 | `{ "error": "upload_interrupted" }` | Client-provided `hash` does not match the server-computed MD5; file was likely corrupted or truncated in transit. |
| 400 | `{ "error": "file not found" }` | No `file` field in the request |
| 500 | `{ "error": "database error" }` | SQLite error |
| 500 | `{ "error": "failed to write file" }` | Disk write error |

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

### GET /files/list

List all uploaded files.

**Response 200**
```json
{
  "files": [
    {
      "file_id": "d41d8cd98f00b204e9800998ecf8427e",
      "file_name": "My Awesome Song"
    }
  ]
}
```

**Errors**

| Status | Body |
|---|---|
| 500 | `{ "error": "database error" }` |

---

### GET /rooms/list

List all active rooms.

**Response 200**
```json
[
  {
    "room_id": "room-alice",
    "name": "chill vibes",
    "owner_id": "alice",
    "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
    "is_playing": false,
    "position": 0,
    "start_at": 0,
    "current_index": 1,
    "queue": ["aaa111", "d41d8cd98f00b204e9800998ecf8427e", "bbb222"]
  }
]
```

**Errors**

| Status | Body |
|---|---|
| 500 | `{ "error": "database error" }` |

---

## WebSocket

**Endpoint:** `GET /ws?user_id=<id>&username=<name>`

`user_id` is required. `username` is optional (defaults to `user_id`). All messages use the envelope format:

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
| `QUEUE_ADD_FAILED` | Redis error |
| `QUEUE_REMOVE_FAILED` | Index out of range or Redis error |
| `QUEUE_MOVE_FAILED` | Index out of range or Redis error |
| `QUEUE_NEXT_FAILED` | End of queue reached or Redis error |
| `QUEUE_PREV_FAILED` | Beginning of queue reached or Redis error |
| `QUEUE_PLAY_AT_FAILED` | Index out of range, track not in library, or Redis error |

---

### RoomState object

All events that carry room state use this shape:

```json
{
  "room_id": "room-alice",
  "name": "chill vibes",
  "owner_id": "alice",
  "track_hash": "d41d8cd98f00b204e9800998ecf8427e",
  "is_playing": false,
  "position": 0,
  "start_at": 0,
  "current_index": 1,
  "queue": ["aaa111", "d41d8cd98f00b204e9800998ecf8427e", "bbb222"]
}
```

| Field | Description |
|---|---|
| `current_index` | Index into `queue` of the active track. `-1` if not playing from queue. |
| `queue` | Ordered list of `track_hash` strings. Empty array `[]` if no queue. |

---

### room:create

**Send**
```json
{
  "event": "room:create",
  "payload": { "user_id": "alice", "room_name": "chill vibes" }
}
```

**Receive — sender** `room:created` → [RoomState](#roomstate-object)

> `room_id` is derived as `room-{user_id}`. `queue` is `[]`, `current_index` is `-1`.

---

### room:join

**Send**
```json
{
  "event": "room:join",
  "payload": { "user_id": "bob", "room_id": "room-alice" }
}
```

**Receive — sender** `room:joined` → [RoomState](#roomstate-object) (includes current `queue` and `current_index`)

**Broadcast — existing members** `room:member:joined`
```json
{ "event": "room:member:joined", "payload": { "user_id": "bob", "username": "Bobby" } }
```

---

### room:leave

**Send**
```json
{
  "event": "room:leave",
  "payload": { "user_id": "bob", "room_id": "room-alice" }
}
```

**Receive — sender** `room:left`
```json
{ "event": "room:left", "payload": { "room_id": "room-alice" } }
```

**Broadcast — remaining members** `room:member:left`
```json
{ "event": "room:member:left", "payload": { "user_id": "bob", "username": "Bobby" } }
```

---

### room:state:request

**Send**
```json
{ "event": "room:state:request", "payload": { "room_id": "room-alice" } }
```

**Receive — sender** `room:state` → [RoomState](#roomstate-object)

---

### track:set

Sets the current track. Must exist in the file library. Resets playback to paused at position 0. Also updates `current_index` to the track's first position in the queue (or `-1` if not found).

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

**Broadcast — all members** `track:changed` → [RoomState](#roomstate-object)

---

### sync:ntp

Clock sync exchange. Run before playback to calibrate client clock offset.

**Send**
```json
{ "event": "sync:ntp", "payload": { "t1": 1708512000000 } }
```

`t1` — client Unix milliseconds at send time.

**Receive — sender** `sync:ntp:pong`
```json
{
  "event": "sync:ntp:pong",
  "payload": { "t1": 1708512000000, "t2": 1708512000012, "t3": 1708512000013 }
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
  "payload": { "user_id": "alice", "room_id": "room-alice", "position": 42.5 }
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
  "payload": { "user_id": "alice", "room_id": "room-alice", "position": 61.3 }
}
```

`position` — client's current playback position in seconds at pause time.

**Broadcast — all members** `sync:pause`
```json
{ "event": "sync:pause", "payload": { "position": 61.3 } }
```

---

### sync:seek

**Send**
```json
{
  "event": "sync:seek",
  "payload": { "user_id": "alice", "room_id": "room-alice", "position": 120.0 }
}
```

**Broadcast — all members** `sync:seek`
```json
{ "event": "sync:seek", "payload": { "position": 120.0, "start_at": 1708512005300 } }
```

`start_at` is non-zero only when the room was playing — treat same as `sync:play`. If `0`, just seek to `position`.

---

## Queue

The queue is an ordered list of `track_hash` strings. `current_index` tracks which entry is active.

All queue mutation events broadcast `queue:updated` to the whole room.  
Queue navigation events (`next`, `prev`, `play_at`) broadcast `track:changed` with the full [RoomState](#roomstate-object).

---

### queue:add

**Send**
```json
{
  "event": "queue:add",
  "payload": {
    "room_id": "room-alice",
    "track_hash": "aaa111",
    "position": -1
  }
}
```

`position` — index to insert at. `-1` (or any value ≥ queue length) appends to the end.

**Broadcast — all members** `queue:updated`
```json
{
  "event": "queue:updated",
  "payload": { "room_id": "room-alice", "queue": ["aaa111", "bbb222"] }
}
```

---

### queue:remove

**Send**
```json
{
  "event": "queue:remove",
  "payload": { "room_id": "room-alice", "index": 0 }
}
```

**Broadcast — all members** `queue:updated` (same shape as above)

---

### queue:move

Reorder the queue (e.g. drag & drop).

**Send**
```json
{
  "event": "queue:move",
  "payload": { "room_id": "room-alice", "from": 3, "to": 0 }
}
```

**Broadcast — all members** `queue:updated` (same shape as above)

---

### queue:next

Advance to the next track. **Idempotent** — the server only advances if its stored `current_index` matches `from_index`. Safe for concurrent presses.

**Send**
```json
{
  "event": "queue:next",
  "payload": { "room_id": "room-alice", "from_index": 1 }
}
```

**Broadcast — all members** `track:changed` → [RoomState](#roomstate-object)

> If already advanced (index mismatch), the server returns the current state as-is with no error.

---

### queue:prev

Go back one track. Same idempotency guarantee as `queue:next`.

**Send**
```json
{
  "event": "queue:prev",
  "payload": { "room_id": "room-alice", "from_index": 2 }
}
```

**Broadcast — all members** `track:changed` → [RoomState](#roomstate-object)

---

### queue:play_at

Jump to a specific queue index directly.

**Send**
```json
{
  "event": "queue:play_at",
  "payload": { "room_id": "room-alice", "index": 3 }
}
```

**Broadcast — all members** `track:changed` → [RoomState](#roomstate-object)

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
   |                                 |<-- WS /ws?user_id=bob---------|
   |                                 |<-- room:join ------------------|
   |<-- room:member:joined -----------|                               |
   |                                 |-- room:joined (with queue) --->|
   |                                 |                                |
   |-- queue:add (track 1) -------->|                                |
   |<-- queue:updated ---------------|-- queue:updated -------------->|
   |-- queue:add (track 2) -------->|                                |
   |<-- queue:updated ---------------|-- queue:updated -------------->|
   |                                 |                                |
   |-- queue:play_at (index: 0) ---->|                                |
   |<-- track:changed ---------------|-- track:changed -------------->|
   |                                 |                                |
   |-- sync:ntp (x3) --------------->|                                |
   |<-- sync:ntp:pong (x3) ----------|                                |
   |                                 |                                |
   |-- sync:play ------------------->|                                |
   |<-- sync:play -------------------|-- sync:play ------------------>|
   |                                 |                                |
   |-- queue:next (from_index: 0) -->|                                |
   |<-- track:changed ---------------|-- track:changed -------------->|
```
