# WebSocket Specification

## 1. Purpose

WebSocket is the real-time transport for room lifecycle and gameplay.

HTTP is used to enter a room and obtain the WebSocket URL. After connection, gameplay commands use WebSocket messages.

## 2. Endpoint

```text
GET /ws/v1/rooms/{room_id}?token=<access_token>
```

Preferred production option:

- Use `Authorization` header during WebSocket upgrade when supported.
- Query token is acceptable for browser MVP, but must not be logged.

## 3. Protocol Envelope

All messages use JSON.

### 3.1 Client To Server

```json
{
  "type": "room.ready",
  "request_id": "req_123",
  "seq": 1,
  "payload": {}
}
```

Fields:

- `type`: message type.
- `request_id`: client-generated request ID.
- `seq`: monotonically increasing client sequence for diagnostics.
- `payload`: message-specific payload.

### 3.2 Server To Client

```json
{
  "type": "ack",
  "request_id": "req_123",
  "seq": 10,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "ok": true
  }
}
```

Fields:

- `type`: message type.
- `request_id`: copied from client message if responding.
- `seq`: server sequence.
- `server_time`: ISO timestamp.
- `payload`: message-specific payload.

## 4. Connection Lifecycle

### 4.1 On Connect

Server must:

1. Validate token.
2. Validate user belongs to room or can observe only if spectator mode exists.
3. Bind connection to user and room.
4. Send room snapshot to the connected user.
5. Broadcast presence change to other players.

First server message:

```json
{
  "type": "room.snapshot",
  "request_id": null,
  "seq": 1,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "room": {},
    "game": {},
    "me": {}
  }
}
```

### 4.2 Heartbeat

Client sends:

```json
{
  "type": "ping",
  "request_id": "req_ping_1",
  "seq": 2,
  "payload": {
    "client_time": "2026-06-03T09:00:01Z"
  }
}
```

Server replies:

```json
{
  "type": "pong",
  "request_id": "req_ping_1",
  "seq": 2,
  "server_time": "2026-06-03T09:00:01Z",
  "payload": {}
}
```

Heartbeat interval:

- Client sends every 15 seconds.
- Server considers connection stale after 45 seconds without pong/read.

### 4.3 Disconnect

Server must:

- Mark player connection offline.
- Keep seat during active game.
- Broadcast player offline event.
- Start reconnect TTL.

## 5. Room Message Types

### 5.1 room.ready

Client:

```json
{
  "type": "room.ready",
  "request_id": "req_1",
  "seq": 3,
  "payload": {
    "ready": true
  }
}
```

Server event:

```json
{
  "type": "room.player_ready",
  "request_id": "req_1",
  "seq": 4,
  "server_time": "2026-06-03T09:00:02Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "ready": true
  }
}
```

### 5.2 room.chat_emote

Optional MVP.

```json
{
  "type": "room.chat_emote",
  "request_id": "req_2",
  "seq": 4,
  "payload": {
    "emote": "hello"
  }
}
```

## 6. Game Message Types

### 6.1 game.bid

Client:

```json
{
  "type": "game.bid",
  "request_id": "req_bid_1",
  "seq": 5,
  "payload": {
    "score": 1
  }
}
```

Allowed scores:

- `0`: pass/no bid.
- `1`
- `2`
- `3`

Server event:

```json
{
  "type": "game.bid_placed",
  "request_id": "req_bid_1",
  "seq": 11,
  "server_time": "2026-06-03T09:00:05Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "score": 1,
    "next_seat_index": 1,
    "deadline_at": "2026-06-03T09:00:20Z"
  }
}
```

### 6.2 game.play_cards

Client:

```json
{
  "type": "game.play_cards",
  "request_id": "req_play_1",
  "seq": 8,
  "payload": {
    "cards": ["S3", "H3", "D3"]
  }
}
```

Server event to all:

```json
{
  "type": "game.cards_played",
  "request_id": "req_play_1",
  "seq": 20,
  "server_time": "2026-06-03T09:01:00Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "cards": ["S3", "H3", "D3"],
    "card_group": {
      "type": "triple",
      "rank": 3,
      "length": 3
    },
    "remaining_count": 14,
    "next_seat_index": 1,
    "deadline_at": "2026-06-03T09:01:15Z"
  }
}
```

Server private event to actor:

```json
{
  "type": "game.my_hand_updated",
  "request_id": "req_play_1",
  "seq": 21,
  "server_time": "2026-06-03T09:01:00Z",
  "payload": {
    "cards": ["S4", "H4"]
  }
}
```

### 6.3 game.pass

Client:

```json
{
  "type": "game.pass",
  "request_id": "req_pass_1",
  "seq": 9,
  "payload": {}
}
```

Server event:

```json
{
  "type": "game.player_passed",
  "request_id": "req_pass_1",
  "seq": 22,
  "server_time": "2026-06-03T09:01:10Z",
  "payload": {
    "user_id": "u_def",
    "seat_index": 1,
    "next_seat_index": 2,
    "deadline_at": "2026-06-03T09:01:25Z"
  }
}
```

### 6.4 game.hint

MVP can implement hint locally on frontend or via server.

Server option:

```json
{
  "type": "game.hint",
  "request_id": "req_hint_1",
  "seq": 10,
  "payload": {}
}
```

Response:

```json
{
  "type": "game.hint_result",
  "request_id": "req_hint_1",
  "seq": 23,
  "server_time": "2026-06-03T09:01:12Z",
  "payload": {
    "suggestions": [
      ["S5"],
      ["S6", "H6"]
    ]
  }
}
```

## 7. Server Broadcast Types

### 7.1 room.snapshot

Snapshot is player-specific.

```json
{
  "type": "room.snapshot",
  "request_id": null,
  "seq": 30,
  "server_time": "2026-06-03T09:01:20Z",
  "payload": {
    "room": {
      "room_id": "r_abc",
      "mode": "classic",
      "status": "playing",
      "base_score": 1
    },
    "players": [
      {
        "user_id": "u_1",
        "display_name": "A",
        "seat_index": 0,
        "role": "landlord",
        "status": "online",
        "ready": true,
        "remaining_count": 17,
        "is_robot": false
      }
    ],
    "game": {
      "game_id": "g_abc",
      "phase": "playing",
      "current_seat_index": 0,
      "landlord_seat_index": 0,
      "bottom_cards": ["BJ", "S2", "H8"],
      "last_play": null,
      "multiplier": 3,
      "deadline_at": "2026-06-03T09:01:35Z"
    },
    "me": {
      "user_id": "u_1",
      "seat_index": 0,
      "hand": ["S3", "H3"]
    }
  }
}
```

Security:

- Only `me.hand` contains private cards.
- Other players expose `remaining_count`, never full hand.

### 7.2 game.started

```json
{
  "type": "game.started",
  "request_id": null,
  "seq": 5,
  "server_time": "2026-06-03T09:00:03Z",
  "payload": {
    "game_id": "g_abc",
    "phase": "bidding",
    "current_seat_index": 0,
    "deadline_at": "2026-06-03T09:00:18Z",
    "my_hand": ["S3", "H3"]
  }
}
```

This message must be sent privately or include player-specific payload.

### 7.3 game.landlord_decided

```json
{
  "type": "game.landlord_decided",
  "request_id": null,
  "seq": 15,
  "server_time": "2026-06-03T09:00:40Z",
  "payload": {
    "landlord_seat_index": 1,
    "landlord_user_id": "u_def",
    "bottom_cards": ["BJ", "S2", "H8"],
    "multiplier": 3,
    "current_seat_index": 1,
    "deadline_at": "2026-06-03T09:00:55Z"
  }
}
```

Landlord also receives `game.my_hand_updated` with bottom cards included.

### 7.4 game.ended

```json
{
  "type": "game.ended",
  "request_id": null,
  "seq": 80,
  "server_time": "2026-06-03T09:08:00Z",
  "payload": {
    "winner_side": "landlord",
    "winner_user_id": "u_def",
    "settlements": [
      {
        "user_id": "u_def",
        "seat_index": 1,
        "role": "landlord",
        "score_delta": 24
      }
    ],
    "final_multiplier": 12,
    "reason": {
      "base_score": 1,
      "bid_score": 3,
      "bomb_count": 1,
      "rocket_count": 0
    }
  }
}
```

## 8. Ack And Error

### 8.1 Ack

```json
{
  "type": "ack",
  "request_id": "req_1",
  "seq": 100,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "ok": true
  }
}
```

Ack is optional if an event with same request ID is immediately emitted.

### 8.2 Error

```json
{
  "type": "error",
  "request_id": "req_play_1",
  "seq": 101,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "code": "invalid_card_set",
    "message": "invalid card set"
  }
}
```

## 9. Ordering And Idempotency

- Server sequence must be monotonically increasing per connection.
- Room event sequence must be monotonically increasing per game.
- Client request IDs should be idempotency hints.
- Duplicate request IDs within a short window should return previous result or reject as duplicate.

## 10. Timeout Handling

Server emits:

```json
{
  "type": "game.timeout",
  "request_id": null,
  "seq": 40,
  "server_time": "2026-06-03T09:02:00Z",
  "payload": {
    "seat_index": 2,
    "action": "auto_pass"
  }
}
```

Rules:

- Bidding timeout defaults to pass or lowest legal action.
- Playing timeout defaults to pass if pass is legal.
- If pass is not legal, robot strategy chooses a minimal legal play.

## 11. Acceptance Criteria

- Client can reconnect and receive `room.snapshot`.
- Invalid turn action returns `error`.
- Other players never receive `me.hand`.
- All state-changing messages are validated by server room actor.
- Timeout events are visible to all players.

