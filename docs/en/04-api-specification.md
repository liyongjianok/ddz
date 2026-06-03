# HTTP API Specification

## 1. API Principles

- HTTP APIs handle authentication, lobby, room metadata, records, and admin-lite queries.
- Real-time gameplay actions use WebSocket, not HTTP.
- All protected endpoints require Bearer JWT.
- Response format must be consistent.
- Server must never expose hidden cards through HTTP.

## 2. Base URL

```text
/api/v1
```

## 3. Common Headers

```text
Authorization: Bearer <access_token>
Content-Type: application/json
X-Request-ID: <uuid optional>
```

## 4. Common Response

### 4.1 Success

```json
{
  "code": "ok",
  "message": "ok",
  "data": {},
  "request_id": "req_123"
}
```

### 4.2 Error

```json
{
  "code": "room_not_found",
  "message": "room not found",
  "data": null,
  "request_id": "req_123"
}
```

## 5. Error Codes

```text
ok
bad_request
unauthorized
forbidden
not_found
rate_limited
internal_error
room_not_found
seat_unavailable
game_already_started
already_in_room
not_in_room
```

## 6. Authentication APIs

### 6.1 Guest Login

```text
POST /api/v1/auth/guest
```

Request:

```json
{
  "display_name": "Guest123",
  "avatar_url": ""
}
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "user": {
      "id": "u_abc",
      "display_name": "Guest123",
      "avatar_url": "",
      "account_type": "guest"
    },
    "access_token": "jwt",
    "expires_in": 86400
  },
  "request_id": "req_123"
}
```

Rules:

- If display name is empty, server generates one.
- Guest accounts are persisted so game records remain stable.

### 6.2 Me

```text
GET /api/v1/auth/me
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "id": "u_abc",
    "display_name": "Guest123",
    "avatar_url": "",
    "account_type": "guest",
    "profile": {
      "level": 1,
      "coin_balance": 10000,
      "total_games": 0,
      "wins": 0
    }
  },
  "request_id": "req_123"
}
```

## 7. Lobby APIs

### 7.1 Lobby Summary

```text
GET /api/v1/lobby/summary
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "online_players": 123,
    "active_rooms": 45,
    "modes": [
      {
        "mode": "classic",
        "base_score": 1,
        "online_players": 100,
        "waiting_rooms": 8
      }
    ]
  },
  "request_id": "req_123"
}
```

### 7.2 Room List

```text
GET /api/v1/rooms?mode=classic&status=waiting&page=1&page_size=20
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "items": [
      {
        "room_id": "r_abc",
        "mode": "classic",
        "status": "waiting",
        "base_score": 1,
        "player_count": 2,
        "max_players": 3,
        "created_at": "2026-06-03T09:00:00Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "request_id": "req_123"
}
```

### 7.3 Quick Start

```text
POST /api/v1/matchmaking/quick-start
```

Request:

```json
{
  "mode": "classic",
  "base_score": 1
}
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 0,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

Rules:

- If user is already in an active room, return that room.
- Matchmaking must not put one user into two rooms.

### 7.4 Create Room

```text
POST /api/v1/rooms
```

Request:

```json
{
  "mode": "classic",
  "base_score": 1,
  "private": false
}
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 0,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

### 7.5 Join Room

```text
POST /api/v1/rooms/{room_id}/join
```

Request:

```json
{
  "preferred_seat": null
}
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 2,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

Rules:

- Cannot join a closed room.
- Cannot join a full room unless replacing an uncommitted pre-game robot.

### 7.6 Leave Room

```text
POST /api/v1/rooms/{room_id}/leave
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "left": true
  },
  "request_id": "req_123"
}
```

Rules:

- Before game starts: user can leave.
- During game: request should mark disconnected or surrender only if surrender feature exists. MVP should not allow HTTP leave to alter active game outcome.

## 8. Record APIs

### 8.1 My Game Records

```text
GET /api/v1/records/my?page=1&page_size=20
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "items": [
      {
        "game_id": "g_abc",
        "mode": "classic",
        "role": "landlord",
        "winner_side": "landlord",
        "score_delta": 24,
        "started_at": "2026-06-03T09:00:00Z",
        "ended_at": "2026-06-03T09:08:00Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "request_id": "req_123"
}
```

### 8.2 Game Record Detail

```text
GET /api/v1/records/{game_id}
```

Response:

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "game_id": "g_abc",
    "room_id": "r_abc",
    "mode": "classic",
    "base_score": 1,
    "multiplier": 12,
    "winner_side": "landlord",
    "players": [
      {
        "user_id": "u_1",
        "display_name": "A",
        "seat_index": 0,
        "role": "landlord",
        "score_delta": 24
      }
    ],
    "events": []
  },
  "request_id": "req_123"
}
```

Security:

- For MVP, only participants can view detailed records.
- Full hidden-card replay is allowed only after game end and only to participants or admins.

## 9. Admin-Lite APIs

These can be protected by admin role and disabled in MVP UI.

### 9.1 Active Rooms

```text
GET /api/v1/admin/rooms/active
```

### 9.2 Game Audit

```text
GET /api/v1/admin/games/{game_id}/audit
```

## 10. HTTP Acceptance Criteria

- Guest login returns valid token.
- Protected APIs reject missing or invalid token.
- Quick start returns a room and seat.
- Room list never includes hidden card data.
- Record detail only returns authorized data.
- Errors follow common response format.

