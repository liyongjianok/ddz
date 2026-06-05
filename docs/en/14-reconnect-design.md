# Reconnect Design

## 1. Purpose

Reconnect allows players to recover from:

- Browser refresh.
- Temporary network loss.
- WebSocket disconnect.
- Mobile sleep/wake.

The server remains authoritative. Reconnect always restores from server state.

## 2. Requirements

MVP requirements:

- Keep player seat during active game.
- Mark player offline on disconnect.
- Allow reconnect within `RECONNECT_TTL`.
- Send player-specific room snapshot after reconnect.
- Preserve private hand only for rightful user.
- Continue game with timeout automation if player is offline.

Default TTL:

```text
RECONNECT_TTL=5m
```

## 3. Connection Identity

Connection is identified by:

- authenticated user ID.
- room ID.
- connection ID.

One active connection per user per room is recommended.

If a second connection appears:

- Prefer latest connection.
- Close old connection with reason `replaced_by_new_connection`.

## 4. State Model

Player connection states:

- `online`
- `offline`
- `reconnecting`

Room seat status should remain:

- `playing` for active game.
- `ready` or `joined` before game.

Do not remove player from active game on disconnect.

## 5. Redis Reconnect Data

Recommended key:

```text
reconnect:{user_id}:{room_id}
```

Value:

```json
{
  "user_id": "u_abc",
  "room_id": "r_abc",
  "seat_index": 0,
  "game_id": "g_abc",
  "disconnected_at": "2026-06-03T09:00:00Z",
  "expires_at": "2026-06-03T09:05:00Z"
}
```

TTL:

- Equal to reconnect TTL.

MVP in-memory alternative:

- Store reconnect metadata in room manager map.
- This does not survive process restart.

## 6. Disconnect Flow

1. WebSocket read/write loop detects close or heartbeat failure.
2. Gateway sends `Disconnect(user_id)` command to room actor.
3. Room actor marks player offline.
4. Room actor emits `player_disconnected`.
5. Reconnect metadata is stored.
6. If it is player's turn, timeout system continues normally.

Broadcast:

```json
{
  "type": "room.player_disconnected",
  "request_id": null,
  "seq": 30,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "reconnect_ttl_seconds": 300
  }
}
```

## 7. Reconnect Flow

1. Client opens WebSocket with same room ID and valid token.
2. Gateway validates token.
3. Gateway checks user has seat in room.
4. Gateway replaces old connection if still present.
5. Gateway sends `Reconnect(user_id)` command to room actor.
6. Room actor marks player online.
7. Room actor emits `player_reconnected`.
8. Server sends player-specific `room.snapshot`.

Private snapshot must include:

- Own hand.
- Current phase.
- Current turn.
- Deadline.
- Last play.
- Bottom cards if already revealed.
- Settlement if game ended.

## 8. Client Reconnect Behavior

Client states:

- `connected`.
- `reconnecting`.
- `resyncing`.
- `failed`.

Algorithm:

1. On WebSocket close, show reconnecting indicator.
2. Retry with exponential backoff:
   - 1s.
   - 2s.
   - 4s.
   - 8s.
   - max 10s.
3. After socket opens, wait for `room.snapshot`.
4. Replace local room state with snapshot.
5. Clear selected cards.

Do not apply stale queued actions after reconnect unless explicitly supported with idempotency.

## 9. Timeout While Offline

Policy:

- Offline player is not immediately removed.
- If their turn times out:
  - Bidding: auto pass.
  - Playing: auto pass if legal; otherwise robot minimal legal play.

If player reconnects after timeout action:

- They see updated state.
- They cannot undo timeout action.

## 10. Game End While Offline

If game ends while player is offline:

- Settlement is persisted.
- Reconnect within TTL returns final settlement snapshot.
- After TTL, user can query record through HTTP API.

## 11. Server Restart

MVP:

- Active games may be lost if only in memory.

Production:

- Store room snapshots in Redis.
- Persist game events incrementally.
- On restart, recover active rooms or mark games aborted according to policy.

## 12. Security

Reconnect must validate:

- JWT token is valid.
- User ID matches seat owner.
- Room exists.
- User is allowed to receive private hand.

Never allow reconnect by only room ID and seat index.

## 13. Acceptance Criteria

- Browser refresh during game restores state.
- Reconnected player sees own current hand.
- Other players see online status change.
- Offline timeout action works.
- Hidden cards are not leaked.

