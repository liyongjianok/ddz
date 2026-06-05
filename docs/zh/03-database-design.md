# 数据库设计

## 1. 数据库目标

数据库用于保存持久业务数据：

- 用户和玩家资料。
- 房间元数据。
- 对局记录。
- 对局事件。
- 结算记录。
- 会话和重连元数据，若不全部放 Redis。

MVP 中活跃对局状态可以保存在内存中；生产环境应定期快照或将重连关键状态放入 Redis。

## 2. 数据库选择

生产推荐：

- PostgreSQL 15+。

原因：

- 强事务能力。
- JSONB 支持灵活事件和快照。
- 索引和运维成熟。

## 3. 命名规范

- 表名使用复数 snake_case。
- 主键统一为 `id`。
- 业务标识使用 `user_id`、`room_id`、`game_id`。
- 时间字段使用 `created_at`、`updated_at`、`deleted_at`。
- JSONB 字段必要时使用 `_json` 后缀。

## 4. 核心表

### 4.1 users

用户身份表。

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

约束：

- `account_type`：`guest`、`registered`、`robot`。
- 游客可以没有 username 和 password。
- 机器人用户必须可被统计区分。

索引：

```sql
CREATE INDEX idx_users_account_type ON users(account_type);
CREATE INDEX idx_users_status ON users(status);
```

### 4.2 player_profiles

玩家资料和统计。

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

MVP 只使用虚拟分/虚拟金币，不包含真金钱语义。

### 4.3 rooms

房间元数据表，不要求保存每次活跃状态变化。

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

状态：

- `waiting`
- `playing`
- `settling`
- `closed`

索引：

```sql
CREATE INDEX idx_rooms_status_mode ON rooms(status, mode);
CREATE INDEX idx_rooms_updated_at ON rooms(updated_at);
```

### 4.4 room_players

房间座位成员表。

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

状态：

- `joined`
- `ready`
- `playing`
- `offline`
- `left`

角色：

- `landlord`
- `farmer`
- 地主未确定前为空。

### 4.5 games

一局完整或进行中的对局。

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

状态：

- `dealing`
- `bidding`
- `playing`
- `ended`
- `aborted`

胜利方：

- `landlord`
- `farmers`

索引：

```sql
CREATE INDEX idx_games_room_id ON games(room_id);
CREATE INDEX idx_games_status ON games(status);
CREATE INDEX idx_games_started_at ON games(started_at);
```

### 4.6 game_players

对局内玩家快照。

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

安全要求：

- `initial_hand_json` 和 `final_hand_json` 是敏感数据。
- 普通玩家 API 不得在无权限情况下暴露。

### 4.7 game_events

追加式事件表。

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

事件类型：

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

索引：

```sql
CREATE INDEX idx_game_events_game_seq ON game_events(game_id, seq);
CREATE INDEX idx_game_events_type ON game_events(event_type);
```

### 4.8 settlements

结算记录表。

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

若 JWT 完全无状态可选；用于吊销和重连追踪时建议保留。

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

## 5. Redis Key 设计

```text
session:{token_id} -> 用户会话 JSON
player_conn:{user_id} -> WebSocket 连接元数据
user_room:{user_id} -> room_id
room_owner:{room_id} -> node_id
room_snapshot:{room_id} -> 简化状态 JSON
reconnect:{user_id}:{room_id} -> 重连元数据
rate_limit:{scope}:{key} -> 限流计数器
```

TTL：

- `session:*`：token TTL。
- `player_conn:*`：连接 TTL，断开时删除。
- `reconnect:*`：`RECONNECT_TTL`，默认 300 秒。
- `room_snapshot:*`：活跃房间 TTL，每次更新刷新。

## 6. JSON 示例

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

## 7. 迁移策略

- 所有 schema 变更必须有 migration。
- 不得修改已在生产应用过的 migration。
- 文件命名：

```text
000001_init.up.sql
000001_init.down.sql
000002_add_game_events.up.sql
000002_add_game_events.down.sql
```

推荐工具：

- `golang-migrate/migrate`。

## 8. 数据保留

MVP：

- 本地和开发环境永久保留。

生产建议：

- 对局记录热存储 180 天。
- 审计日志热存储 90 天。
- 聚合玩家统计永久保留。
- 旧事件可归档到对象存储。

## 9. 数据库验收标准

- 对局完成后写入 `games`、`game_players`、`game_events`、`settlements`。
- 玩家统计和结算在事务内更新。
- 大厅和房间列表不暴露隐藏牌。
- 对局事件可重建一局已完成游戏。
- schema 支持机器人和游客用户。

