# Production Architecture

## 1. Production Goal

Scale from MVP single-process deployment to a production棋牌游戏-style architecture:

- Reliable WebSocket connectivity.
- Server-authoritative room state.
- Horizontally scalable APIs.
- Observable game events.
- Recoverable room state.
- Clear separation of gameplay, persistence, and operations.

## 2. MVP vs Production

### 2.1 MVP

```text
Browser -> Single Go Server -> PostgreSQL
                         -> Redis optional
```

Characteristics:

- HTTP and WebSocket in one process.
- Active rooms in memory.
- Game records persisted on completion.
- Simple robot service in same process.

### 2.2 Production

```text
Browser
  -> CDN / Reverse Proxy
  -> API Service
  -> WebSocket Gateway
  -> Room Service / Room Actors
  -> PostgreSQL / Redis / Message Broker
```

Characteristics:

- HTTP and WebSocket can scale independently.
- Room ownership registry.
- Redis for sessions, reconnect, and active snapshots.
- Event persistence and analytics pipeline.

## 3. Production Components

### 3.1 API Service

Responsibilities:

- Auth.
- Lobby.
- Matchmaking entry.
- Records.
- Admin-lite APIs.

Should be stateless except database/Redis dependencies.

### 3.2 WebSocket Gateway

Responsibilities:

- Connection upgrade.
- Token validation.
- Heartbeat.
- Message schema validation.
- Connection mapping.
- Forward commands to owning room actor.
- Send events to clients.

Scaling:

- Sticky routing by user ID or room ID is helpful.
- Cross-node routing needs Redis pub/sub, NATS, or similar.

### 3.3 Room Service

Responsibilities:

- Room actor lifecycle.
- Serialized command handling.
- Game state.
- Timers.
- Robot command integration.
- Snapshot generation.

Scaling strategy:

- Assign each active room to one node.
- Store `room_owner:{room_id} -> node_id` in Redis.
- Commands for remote rooms are routed to owner node.

### 3.4 Rules Engine

Responsibilities:

- Card parsing.
- Group recognition.
- Comparison.
- Legal move generation.
- Settlement.

Must remain pure library code.

### 3.5 Robot Service

MVP:

- In-process service.

Production:

- Can remain in-process or move to separate worker pool.
- Must submit actions through room command protocol.

### 3.6 Persistence Service

Responsibilities:

- Game event writes.
- Settlement writes.
- Player stats.
- Record query.

Use transactions for settlement.

## 4. Data Stores

### 4.1 PostgreSQL

Stores:

- Users.
- Profiles.
- Rooms metadata.
- Games.
- Game events.
- Settlements.
- Audit logs.

### 4.2 Redis

Stores:

- Session cache.
- WebSocket connection metadata.
- User active room mapping.
- Room owner mapping.
- Active room snapshots.
- Reconnect metadata.
- Rate limit counters.

### 4.3 Message Broker

Optional for MVP, recommended for production:

- NATS.
- Redis Streams.
- Kafka for analytics-heavy future.

Use cases:

- Cross-node room commands.
- Event fanout.
- Analytics ingestion.

## 5. Scaling Model

### 5.1 Room Ownership

Only one node owns a room at a time.

Ownership key:

```text
room_owner:{room_id} = node_id
```

Requirements:

- Owner periodically renews lease.
- If owner dies, room recovery process uses latest snapshot/events.

### 5.2 Sticky Sessions

Options:

1. Route all room WebSocket connections to owner node.
2. Allow gateway node to forward commands/events to room owner.

MVP chooses option 1 by running one node.

Production should prepare for option 2.

## 6. Reliability

### 6.1 Graceful Shutdown

On shutdown:

1. Mark node draining.
2. Stop accepting new rooms.
3. Notify clients where possible.
4. Persist room snapshots.
5. Finish active command processing.
6. Release room ownership leases.

### 6.2 Room Recovery

Recovery inputs:

- Last room snapshot from Redis.
- Persisted game events.
- Reconnect metadata.

Recovery policies:

- Recover if snapshot is fresh.
- Abort and settle as no-contest if state cannot be recovered safely.

### 6.3 Event Durability

Recommended:

- Persist critical events during game, not only at end.
- At minimum persist game start and game end.
- Production should append events as actions occur.

## 7. Security And Anti-Cheat

### 7.1 Server Authority

Server owns:

- Shuffle.
- Deal.
- Hands.
- Turn.
- Move validation.
- Settlement.

### 7.2 Hidden Information

Protection:

- Player-specific snapshots.
- Strict DTO separation.
- No full-state broadcast.
- Log redaction.

### 7.3 Rate Limiting

Rate limit:

- Login attempts.
- HTTP APIs.
- WebSocket messages per connection.
- Invalid gameplay actions.

### 7.4 Audit

Audit suspicious:

- Excessive invalid moves.
- Frequent disconnects on bad hands.
- Multiple accounts from same IP.
- Abnormal win rates.

MVP logs only; future can add scoring.

## 8. Observability

### 8.1 Metrics

Recommended metrics:

- Online users.
- Active rooms.
- WebSocket connections.
- Messages per second.
- Invalid actions per minute.
- Room actor queue length.
- Command processing latency.
- Reconnect success rate.
- Game completion count.
- Average game duration.

### 8.2 Logs

Structured logs:

- trace ID.
- user ID.
- room ID.
- game ID.
- event type.
- severity.

### 8.3 Tracing

Trace:

- HTTP request.
- WebSocket command handling.
- Room actor command.
- DB write.

## 9. Performance Targets

MVP:

- 1,000 idle WebSocket connections per node in dev benchmark.
- 100 active rooms per node.
- Rule validation under 5 ms.

Production initial:

- 10,000 idle WebSocket connections per gateway node, depending on instance size.
- 1,000 active rooms per room-service node, depending on timer and robot load.
- P95 command processing under 100 ms.

Actual targets must be verified with load testing.

## 10. Release Strategy

### 10.1 Environments

- local.
- dev.
- staging.
- production.

### 10.2 Deployment Flow

1. Run unit tests.
2. Run integration tests.
3. Build Docker images.
4. Apply migrations.
5. Deploy staging.
6. Run smoke tests.
7. Deploy production.
8. Monitor metrics/logs.

## 11. Production Acceptance Criteria

- WebSocket works behind reverse proxy.
- Horizontal API scaling does not break auth/lobby.
- Room state does not mutate concurrently across nodes.
- Reconnect works within TTL.
- Game records are durable.
- Hidden cards are not leaked in logs, APIs, or broadcasts.
- Graceful shutdown does not corrupt completed games.

