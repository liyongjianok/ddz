# Database Design

## 1. Database Goals

The database stores durable business data:

- Users and player profile.
- Rooms metadata.
- Game records.
- Game events.
- Settlement records.
- Session and reconnect metadata if not stored only in Redis.

Active in-progress game state may be kept in memory for MVP, but production should periodically snapshot it or keep reconnect-critical state in Redis.

## 2. Database Choice

Recommended production database:

- PostgreSQL 15+.

Reasons:

- Strong transactions.
- JSONB support for flexible game snapshots/events.
- Good indexing and operational maturity.

## 3. Naming Conventions

- Table names: plural snake_case.
- Primary key: `id`.
- Business identifiers: `user_id`, `room_id`, `game_id`.
- Timestamps: `created_at`, `updated_at`, `deleted_at`.
- JSONB fields: suffix with `_json` only when clarity is needed.

## 4. Core Tables

### 4.1 users

Stores account identity.

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(36) NOT NULL UNIQUE,
    username VARCHAR(64),
    display_name VARCHAR(64) NOT NULL,
    avatar_url TEXT,
    account_type VARCHAR(20) NOT NULL,
    password_hash TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Constraints:

- `account_type` values: `guest`, `registered`, `robot`.
- Guest users may not have username or password.
- Robot users should be distinguishable for analytics.

Indexes:

```sql
CREATE INDEX idx_users_account_type ON users(account_type);
CREATE INDEX idx_users_status ON users(status);
```

### 4.2 player_profiles

Stores gameplay profile and stats.

```sql
CREATE TABLE player_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    level INT NOT NULL DEFAULT 1,
    exp BIGINT NOT NULL DEFAULT 0,
    coin_balance BIGINT NOT NULL DEFAULT 10000,
    total_games BIGINT NOT NULL DEFAULT 0,
    wins BIGINT NOT NULL DEFAULT 0,
    landlord_games BIGINT NOT NULL DEFAULT 0,
    landlord_wins BIGINT NOT NULL DEFAULT 0,
    farmer_games BIGINT NOT NULL DEFAULT 0,
    farmer_wins BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

MVP uses virtual score/coins only. No real-money semantics.

### 4.3 rooms

Stores room metadata, not necessarily every active state mutation.

```sql
CREATE TABLE rooms (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(36) NOT NULL UNIQUE,
    mode VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    base_score INT NOT NULL DEFAULT 1,
    max_players INT NOT NULL DEFAULT 3,
    owner_user_id BIGINT REFERENCES users(id),
    current_game_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ
);
```

Status values:

- `waiting`
- `playing`
- `settling`
- `closed`

Indexes:

```sql
CREATE INDEX idx_rooms_status_mode ON rooms(status, mode);
CREATE INDEX idx_rooms_updated_at ON rooms(updated_at);
```

### 4.4 room_players

Tracks seat membership.

```sql
CREATE TABLE room_players (
    id BIGSERIAL PRIMARY KEY,
    room_id BIGINT NOT NULL REFERENCES rooms(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    seat_index INT NOT NULL,
    role VARCHAR(20),
    status VARCHAR(20) NOT NULL,
    is_robot BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    UNIQUE(room_id, seat_index),
    UNIQUE(room_id, user_id)
);
```

Status values:

- `joined`
- `ready`
- `playing`
- `offline`
- `left`

Role values:

- `landlord`
- `farmer`
- null before role is assigned.

### 4.5 games

Stores one complete or in-progress match.

```sql
CREATE TABLE games (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(36) NOT NULL UNIQUE,
    room_id BIGINT NOT NULL REFERENCES rooms(id),
    mode VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    base_score INT NOT NULL,
    multiplier INT NOT NULL DEFAULT 1,
    landlord_user_id BIGINT REFERENCES users(id),
    winner_side VARCHAR(20),
    shuffle_seed_hash VARCHAR(128),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Status values:

- `dealing`
- `bidding`
- `playing`
- `ended`
- `aborted`

Winner side:

- `landlord`
- `farmers`

Indexes:

```sql
CREATE INDEX idx_games_room_id ON games(room_id);
CREATE INDEX idx_games_status ON games(status);
CREATE INDEX idx_games_started_at ON games(started_at);
```

### 4.6 game_players

Stores player snapshot for a game.

```sql
CREATE TABLE game_players (
    id BIGSERIAL PRIMARY KEY,
    game_id BIGINT NOT NULL REFERENCES games(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    seat_index INT NOT NULL,
    role VARCHAR(20),
    is_robot BOOLEAN NOT NULL DEFAULT false,
    initial_hand_json JSONB NOT NULL,
    final_hand_json JSONB,
    score_delta BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(game_id, user_id),
    UNIQUE(game_id, seat_index)
);
```

Security:

- `initial_hand_json` and `final_hand_json` are sensitive.
- Do not expose through normal player-facing APIs unless authorized for own game replay after game end.

### 4.7 game_events

Append-only event log.

```sql
CREATE TABLE game_events (
    id BIGSERIAL PRIMARY KEY,
    game_id BIGINT NOT NULL REFERENCES games(id),
    seq INT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    actor_user_id BIGINT REFERENCES users(id),
    payload_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(game_id, seq)
);
```

Event types:

- `game_started`
- `cards_dealt`
- `bid_placed`
- `landlord_decided`
- `bottom_cards_revealed`
- `cards_played`
- `player_passed`
- `turn_changed`
- `timeout_triggered`
- `player_disconnected`
- `player_reconnected`
- `game_ended`
- `settlement_completed`

Indexes:

```sql
CREATE INDEX idx_game_events_game_seq ON game_events(game_id, seq);
CREATE INDEX idx_game_events_type ON game_events(event_type);
```

### 4.8 settlements

Stores score changes.

```sql
CREATE TABLE settlements (
    id BIGSERIAL PRIMARY KEY,
    game_id BIGINT NOT NULL REFERENCES games(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL,
    base_score INT NOT NULL,
    multiplier INT NOT NULL,
    score_delta BIGINT NOT NULL,
    reason_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(game_id, user_id)
);
```

### 4.9 user_sessions

Optional if JWT is stateless. Useful for revocation and reconnect tracking.

```sql
CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    token_id VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    user_agent TEXT,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
```

### 4.10 audit_logs

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    trace_id VARCHAR(64),
    user_id BIGINT REFERENCES users(id),
    room_id BIGINT REFERENCES rooms(id),
    game_id BIGINT REFERENCES games(id),
    event_type VARCHAR(64) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    payload_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 5. Redis Keys

Recommended keys:

```text
session:{token_id} -> user session JSON
player_conn:{user_id} -> websocket connection metadata
user_room:{user_id} -> room_id
room_owner:{room_id} -> node_id
room_snapshot:{room_id} -> compact state JSON
reconnect:{user_id}:{room_id} -> reconnect metadata
rate_limit:{scope}:{key} -> counter
```

TTL:

- `session:*`: token TTL.
- `player_conn:*`: connection TTL or delete on disconnect.
- `reconnect:*`: `RECONNECT_TTL`, default 300 seconds.
- `room_snapshot:*`: active room TTL, refreshed on updates.

## 6. JSON Payload Examples

### 6.1 game_events.cards_played

```json
{
  "seat_index": 1,
  "cards": ["S3", "H3", "D3"],
  "card_group": {
    "type": "triple",
    "rank": 3,
    "length": 3
  },
  "remaining_count": 12
}
```

### 6.2 settlements.reason_json

```json
{
  "winner_side": "landlord",
  "base_score": 1,
  "multipliers": [
    {"type": "bid", "value": 3},
    {"type": "bomb", "value": 2},
    {"type": "rocket", "value": 2}
  ],
  "final_multiplier": 12
}
```

## 7. Migration Policy

- All schema changes must be represented as migrations.
- Never edit an already-applied production migration.
- Migration filenames:

```text
000001_init.up.sql
000001_init.down.sql
000002_add_game_events.up.sql
000002_add_game_events.down.sql
```

Recommended tool:

- `golang-migrate/migrate`.

## 8. Data Retention

MVP:

- Keep all records indefinitely in local/dev.

Production recommendation:

- Game records: 180 days hot storage.
- Audit logs: 90 days hot storage.
- Aggregated player stats: indefinitely.
- Archive old events to object storage if needed.

## 9. Database Acceptance Criteria

- Game completion writes `games`, `game_players`, `game_events`, and `settlements`.
- Player stats update transactionally with settlement.
- Hidden hands are never exposed through room list or lobby APIs.
- Game events can reconstruct a completed game.
- Schema supports robot and guest users.

