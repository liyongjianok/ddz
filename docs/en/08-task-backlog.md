# Task Backlog

## 1. Sprint Strategy

Develop in small vertical slices. Each sprint should leave the project runnable or at least testable.

Recommended sprint length for AI-assisted development:

- 1 to 3 days per sprint.
- 3 to 8 tightly scoped tasks per sprint.

## 2. Milestone 0: Project Scaffold

### T-0001 Create Repository Structure

Scope:

- Create `backend/`, `frontend/`, `deploy/`, `docs/`.
- Initialize Go module.
- Initialize frontend Vite React TypeScript app.
- Add basic README.

Acceptance:

- `go test ./...` runs.
- frontend install/build command is documented.

### T-0002 Backend Config And Server Skeleton

Scope:

- Add config loader.
- Add HTTP server skeleton.
- Add health endpoint.

Acceptance:

- `GET /healthz` returns OK.
- Config can be loaded from env.

## 3. Milestone 1: Game Rules Engine

### T-1001 Card Model And Deck

Docs:

- [06-domain-model.md](./06-domain-model.md)
- [12-game-rules-engine.md](./12-game-rules-engine.md)

Scope:

- Implement `Card`, `Rank`, `Suit`.
- Parse card code.
- Generate standard deck.
- Shuffle with injectable RNG.
- Deal 17/17/17 plus 3 bottom cards.

Acceptance:

- Deck has 54 unique cards.
- Invalid card codes rejected.
- Deal counts are correct.

### T-1002 Card Group Recognition

Scope:

- Recognize all MVP card groups.
- Return type, primary rank, length, attachments.

Acceptance:

- Unit tests cover every legal and invalid type.

### T-1003 Move Comparison

Scope:

- Implement `CanBeat(candidate, previous)`.
- Handle bomb and rocket.

Acceptance:

- Same pattern higher rank wins.
- Bomb beats non-bomb.
- Rocket beats all.
- Invalid comparisons rejected.

### T-1004 Legal Move Generator

Scope:

- Generate legal responses from hand.
- Generate opening plays.

Acceptance:

- Used by hint and robot.
- Tests cover common hands.

## 4. Milestone 2: Game State Machine

### T-2001 Game State Initialization

Scope:

- Create game with players.
- Shuffle/deal.
- Set bidding start seat.

Acceptance:

- Initial state satisfies invariants.

### T-2002 Bidding State

Scope:

- Implement bid validation.
- Select landlord.
- Handle all-pass.

Acceptance:

- Invalid bid rejected.
- Bid 3 immediately ends bidding.
- Landlord receives bottom cards.

### T-2003 Playing State

Scope:

- Implement play cards.
- Implement pass.
- Advance turns.
- Detect game over.

Acceptance:

- Cannot play out of turn.
- Cannot play missing cards.
- Cannot pass on opening turn.
- Game ends when hand is empty.

### T-2004 Settlement

Scope:

- Compute winner side.
- Compute score deltas.
- Apply multiplier.

Acceptance:

- Landlord win/lose deltas correct.
- Farmer win/lose deltas correct.
- Multiplier reasons returned.

## 5. Milestone 3: Room Runtime

### T-3001 Room Manager

Scope:

- Create room.
- Join room.
- Leave waiting room.
- Find quick-start room.

Acceptance:

- One user cannot join two active rooms.
- Full room rejects extra user.

### T-3002 Room Actor

Scope:

- Serialize room commands.
- Maintain seats and ready states.
- Start game when ready.

Acceptance:

- Concurrent ready/join commands do not corrupt state.

### T-3003 Room Snapshots

Scope:

- Generate player-specific snapshot.

Acceptance:

- Own hand included.
- Other hands excluded.

### T-3004 Timeout System

Scope:

- Turn deadlines.
- Timeout command.
- Auto bid/pass/play.

Acceptance:

- Timeout enters same command queue.
- No duplicate timeout action after user acts.

## 6. Milestone 4: HTTP APIs

### T-4001 Guest Auth

Scope:

- Guest login.
- JWT issue/verify.
- Auth middleware.

Acceptance:

- Protected endpoint rejects invalid token.

### T-4002 Lobby APIs

Scope:

- Lobby summary.
- Room list.

Acceptance:

- Room list contains no hidden cards.

### T-4003 Matchmaking APIs

Scope:

- Quick start.
- Create room.
- Join room.
- Leave room.

Acceptance:

- Returns room ID and WebSocket URL.

## 7. Milestone 5: WebSocket

### T-5001 WebSocket Gateway

Scope:

- Upgrade endpoint.
- Auth token validation.
- Connection lifecycle.

Acceptance:

- Connect sends room snapshot.

### T-5002 Protocol Handling

Scope:

- Decode message envelope.
- Route ready/bid/play/pass/ping.
- Return error messages.

Acceptance:

- Unknown type returns error.
- Invalid action returns stable error code.

### T-5003 Broadcast System

Scope:

- Broadcast public events.
- Send private hand updates.

Acceptance:

- Hidden cards are not leaked.

## 8. Milestone 6: Frontend MVP

### T-6001 Frontend Scaffold

Scope:

- Vite React TypeScript.
- Routing.
- API client.
- Auth store.

Acceptance:

- Guest login works.

### T-6002 Lobby UI

Scope:

- Summary.
- Quick start.
- Room list.

Acceptance:

- User can enter room.

### T-6003 Room UI

Scope:

- Player seats.
- Hand cards.
- Bottom cards.
- Turn indicator.
- Countdown.
- Ready/bid/play/pass controls.

Acceptance:

- Three browser clients can complete game.

### T-6004 Card Interaction

Scope:

- Card sorting.
- Selection.
- Play selected.
- Hint.

Acceptance:

- Selected cards visually clear.
- Illegal action buttons disabled where client can infer.

## 9. Milestone 7: Robot

### T-7001 Robot User And Seat Fill

Scope:

- Create robot user identity.
- Fill empty seats after delay.

Acceptance:

- User can play alone with two robots.

### T-7002 Robot Strategy

Scope:

- Bidding heuristic.
- Play heuristic.
- Pass heuristic.

Acceptance:

- Robot always emits legal actions.

## 10. Milestone 8: Reconnect

### T-8001 Connection Mapping

Scope:

- Track user connection state.
- Mark offline on disconnect.

Acceptance:

- UI shows offline status.

### T-8002 Reconnect Snapshot

Scope:

- Reconnect with token.
- Restore room and private hand.

Acceptance:

- Browser refresh during game resumes.

## 11. Milestone 9: Persistence

### T-9001 Migrations

Scope:

- Create tables from database design.

Acceptance:

- Migration up/down works.

### T-9002 Game Event Persistence

Scope:

- Persist game events.
- Persist settlement.

Acceptance:

- Completed game has queryable record.

### T-9003 Player Stats

Scope:

- Update profile stats after settlement.

Acceptance:

- Stats match completed games.

## 12. Milestone 10: Production Readiness

### T-10001 Docker Compose

Scope:

- Backend.
- Frontend.
- PostgreSQL.
- Redis.
- Reverse proxy optional.

Acceptance:

- Local stack starts with one command.

### T-10002 Observability

Scope:

- Structured logs.
- Basic metrics.
- Health checks.

Acceptance:

- Logs contain trace/user/room/game IDs.

### T-10003 Load Smoke Test

Scope:

- Simulate connections.
- Simulate room actions.

Acceptance:

- 1,000 idle WebSocket connections in dev benchmark target.

