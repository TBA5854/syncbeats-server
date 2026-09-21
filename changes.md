# SyncBeats Backend API Changes

This document outlines the changes in the backend API that the frontend needs to adapt to.

## Room Join Response

The response to joining a room has been updated to include more information about the room's state and its members.

### `POST /api/room/join`

The response body for a successful room join has been updated.

**New Response Body:**

```json
{
    "id": "c1b2a3f4-5d6e-7f8a-9b0c-1d2e3f4a5b6c",
    "name": "The Funky Bunch",
    "host_id": "user-123",
    "users": [
        {
            "id": "user-123",
            "name": "Alice"
        },
        {
            "id": "user-456",
            "name": "Bob"
        }
    ],
    "playback": {
        "track_id": "track-789",
        "track_hash": "a1b2c3d4e5f6...",
        "is_playing": true,
        "started_at": "2026-03-03T10:00:00Z",
        "seek_offset": 123.45
    }
}
```

### Field Descriptions:

-   `id` (string): The room's unique identifier.
-   `name` (string): The room's display name.
-   `host_id` (string): The ID of the user who is the host of the room.
-   `users` (array): A list of users currently in the room.
    -   `id` (string): The user's unique identifier.
    -   `name` (string): The user's display name.
-   `playback` (object): Information about the current playback state.
    -   `track_id` (string): The ID of the current track.
    -   `track_hash` (string): The hash of the current track file.
    if `is_playing` is false, then the track is paused at `seek_offset`. `started_at` should be ignored.
    -   `is_playing` (boolean): Whether the track is currently playing.
    -   `started_at` (string): The authoritative server time (ISO 8601 UTC) when playback started. The client should use this and the current server time to calculate the exact seek position, accounting for network latency.
    -   `seek_offset` (float64): The position in seconds where the track was paused. This is only relevant if `is_playing` is `false`.

## WebSocket Events

The WebSocket events for playback control have been updated.

### `PLAY` Event

-   **Action**: Sent by the host to start playback.
-   **Payload**:
    ```json
    {
        "type": "PLAY",
        "payload": {
            "track_id": "track-789",
            "started_at": "2026-03-03T10:00:00Z"
        }
    }
    ```

### `PAUSE` Event

-   **Action**: Sent by the host to pause playback.
-   **Payload**:
    ```json
    {
        "type": "PAUSE",
        "payload": {
            "seek_offset": 123.45
        }
    }
    ```

### `SEEK` Event

-   **Action**: Sent by the host to seek to a new position.
-   **Payload**:
    ```json
    {
        "type": "SEEK",
        "payload": {
            "started_at": "2026-03-03T10:00:00Z",
            "seek_offset": 60.0
        }
    }
    ```
