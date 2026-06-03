# System Architecture

## 1. Architecture Goal

The system must support a browser-based real-time Dou Dizhu game with clear boundaries:

- HTTP API for login, lobby, room list, records, and admin-lite queries.
- WebSocket gateway for real-time room gameplay.
- Domain rules engine independent from transport and persistence.
- Room actor/service that serializes game actions per room.
- Persistence layer for users, rooms, game records, and events.
- Robot service that can participate like a player through internal commands.

The architecture should be simple enough for MVP but shaped for production growth.

## 2. Recommended Tech Stack

### 2.1 Backend

- Language: Go 1.22+
- HTTP framework: Gin, Echo, or chi. Prefer chi for lightweight routing.
- WebSocket: `nhooyr.io/websocket` or `gorilla/websocket`. Prefer `nhooyr.io/websocket` for context-friendly APIs.
- Database: PostgreSQL 15+ for production, SQLite optional only for local prototype.
- Cache/session: Redis 7+.
- ORM/query: sqlc or GORM. Prefer sqlc for type safety in production.
- Config: env vars plus config file.
- Logging: zerolog or slog. Prefer standard `log/slog` unless project needs advanced features.
- Tests: Go testing, testify optional.

### 2.2 Frontend

- React 18+
- TypeScript
- Vite
- Zustand or Redux Toolkit. Prefer Zustand for MVP.
- Tailwind CSS or CSS Modules. Prefer Tailwind only if already configured cleanly.
- WebSocket client abstraction.

### 2.3 Deployment

- Docker.
- Docker Compose for dev/staging.
- Nginx or Caddy reverse proxy.
- PostgreSQL.
- Redis.

## 3. High-Level Components

```text
Browser Client
  | HTTP
  v
API Server
  | auth/session/db
  v
Database

Browser Client
  | WebSocket
  v
WebSocket Gateway
  | room commands/events
  v
Room Runtime / Game Service
  | rules validation
  v
Rules Engine

Room Runtime
  | events/snapshots
  v
Persistence

Robot Service
  | internal commands
  v
Room Runtime
```

## 4. Backend Package Layout

Recommended Go layout:

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── server.go
│   │   └── config.go
│   ├── auth/
│   │   ├── jwt.go
│   │   ├── service.go
│   │   └── middleware.go
│   ├── user/
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── lobby/
│   │   ├── service.go
│   │   └── handler.go
│   ├── room/
│   │   ├── actor.go
│   │   ├── manager.go
│   │   ├── service.go
│   │   ├── snapshot.go
│   │   └── timeout.go
│   ├── game/
│   │   ├── card.go
│   │   ├── deck.go
│   │   ├── rules.go
│   │   ├── compare.go
│   │   ├── state.go
│   │   ├── event.go
│   │   └── settlement.go
│   ├── robot/
│   │   ├── strategy.go
│   │   ├── evaluator.go
│   │   └── service.go
│   ├── ws/
│   │   ├── gateway.go
│   │   ├── connection.go
│   │   ├── protocol.go
│   │   └── hub.go
│   ├── record/
│   │   ├── repository.go
│   │   └── service.go
│   ├── storage/
│   │   ├── db.go
│   │   ├── redis.go
│   │   └── migrations/
│   └── observability/
│       ├── logging.go
│       └── metrics.go
├── pkg/
│   └── errors/
└── tests/
```

Rules:

- `internal/game` must not depend on HTTP, WebSocket, database, Redis, or frontend DTOs.
- `internal/room` may depend on `internal/game`, persistence interfaces, and robot interfaces.
- `internal/ws` may translate protocol messages into room commands.
- `internal/robot` may read game snapshots but must not mutate game state directly.

## 5. Frontend Layout

```text
frontend/
├── src/
│   ├── app/
│   │   ├── App.tsx
│   │   └── routes.tsx
│   ├── api/
│   │   ├── http.ts
│   │   ├── auth.ts
│   │   ├── lobby.ts
│   │   └── records.ts
│   ├── ws/
│   │   ├── client.ts
│   │   ├── protocol.ts
│   │   └── roomSocket.ts
│   ├── store/
│   │   ├── authStore.ts
│   │   ├── lobbyStore.ts
│   │   └── roomStore.ts
│   ├── domain/
│   │   ├── cards.ts
│   │   ├── gameTypes.ts
│   │   └── viewModels.ts
│   ├── pages/
│   │   ├── LoginPage.tsx
│   │   ├── LobbyPage.tsx
│   │   └── RoomPage.tsx
│   ├── components/
│   │   ├── cards/
│   │   ├── room/
│   │   ├── lobby/
│   │   └── common/
│   └── styles/
└── tests/
```

Frontend rules:

- Client may suggest cards but server validates all moves.
- Client must render from server snapshots/events.
- Client must not infer hidden cards.
- Reconnect should request authoritative snapshot.

## 6. Room Runtime Model

Each room should behave like an actor:

- One command queue per room.
- Commands processed sequentially.
- No concurrent mutation of room game state.
- Emits events after state changes.
- Broadcasts player-specific snapshots.

Command examples:

- JoinRoom.
- LeaveRoom.
- Ready.
- Bid.
- PlayCards.
- Pass.
- Timeout.
- Disconnect.
- Reconnect.

Event examples:

- RoomJoined.
- PlayerReady.
- GameStarted.
- BidPlaced.
- LandlordDecided.
- CardsPlayed.
- PlayerPassed.
- TurnChanged.
- GameEnded.

## 7. State Ownership

### 7.1 Server-Owned State

- Deck.
- Shuffle seed/hash.
- Player hands.
- Current turn.
- Last valid play.
- Bidding history.
- Timeout deadlines.
- Settlement.

### 7.2 Client-Owned State

- Selected cards.
- UI sort preference.
- Local animation state.
- Temporary optimistic visual hints.

Client-owned state must never be used as game truth.

## 8. Persistence Strategy

MVP can keep active room state in memory and persist completed game events.

Production should use:

- Redis for active room registry, player connection mapping, and short-lived reconnect state.
- Database for durable game records.
- Optional event stream for analytics.

Recommended event persistence:

- Persist important game events as append-only records.
- Store final snapshot for fast record display.
- Use transaction boundaries around game completion.

## 9. API Gateway And WebSocket Scaling

MVP:

- Single process handles HTTP and WebSocket.

Production:

- Multiple HTTP instances.
- WebSocket gateway instances with sticky routing by room ID or player ID.
- Redis pub/sub or message broker for cross-node room events.
- Room ownership registry so commands reach the node owning the room.

## 10. Error Handling

Use typed errors:

- `ErrUnauthorized`
- `ErrRoomNotFound`
- `ErrSeatUnavailable`
- `ErrGameAlreadyStarted`
- `ErrNotPlayerTurn`
- `ErrInvalidBid`
- `ErrInvalidCardSet`
- `ErrCannotPass`
- `ErrStateConflict`

Protocol responses should include:

- code.
- message.
- request_id.

Do not expose internal stack traces to clients.

## 11. Configuration

Environment variables:

- `APP_ENV`
- `HTTP_ADDR`
- `DATABASE_URL`
- `REDIS_URL`
- `JWT_SECRET`
- `ACCESS_TOKEN_TTL`
- `ROOM_IDLE_TTL`
- `RECONNECT_TTL`
- `TURN_TIMEOUT_SECONDS`
- `ROBOT_FILL_DELAY_SECONDS`
- `LOG_LEVEL`

## 12. Development Milestones

1. Game rules engine and tests.
2. Room actor with in-memory state.
3. HTTP auth and lobby APIs.
4. WebSocket gateway and protocol.
5. Frontend login/lobby/room.
6. Robot integration.
7. Reconnect.
8. Persistence and records.
9. Deployment and observability.

## 13. Architecture Acceptance Criteria

- Rules engine can be unit-tested without server.
- Room commands are serialized.
- WebSocket messages have stable schemas.
- Hidden cards are never broadcast to other players.
- Completed match is reproducible from persisted events.
- Backend can be started with one command in local dev.

