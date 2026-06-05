# Web Dou Dizhu Product Requirements

## 1. Product Vision

Build a production-grade web Dou Dizhu game that can be developed module by module with AI coding tools. The first release targets a stable, fair, and responsive casual card-room experience:

- Browser-based gameplay.
- Real-time multiplayer rooms.
- Human players plus robot fallback.
- Deterministic game rules engine.
- Reconnect and resume support.
- Auditable game records.
- Extensible architecture for future wallet, ranking, tournament, and anti-cheat systems.

This project must be implemented as a real engineering system, not as a throwaway demo.

## 2. Target Users

### 2.1 Casual Player

- Wants to quickly enter a room and play.
- Expects familiar Dou Dizhu rules.
- Needs clear turn prompts, card selection, countdowns, and result display.

### 2.2 Returning Player

- Wants persistent profile, statistics, and reconnect support.
- Expects previous unfinished match recovery.

### 2.3 Operator/Admin

- Needs observable game rooms and player sessions.
- Needs logs, replay records, and basic fraud investigation data.

### 2.4 AI Coding Agent

- Must be able to implement one module at a time from documentation.
- Must know module boundaries, interfaces, tests, and acceptance criteria.

## 3. Product Scope

### 3.1 MVP Scope

MVP includes:

- User guest login and optional registered account login.
- Lobby with room list and quick start.
- Room creation, matching, ready state.
- Three-player classic Dou Dizhu.
- Landlord bidding/calling phase.
- Playing phase with pass, prompt, play cards.
- Basic robot player.
- Server-authoritative game state.
- WebSocket synchronization.
- Reconnect within a configurable TTL.
- Match settlement with virtual score only.
- Game record persistence.
- Basic admin-observable logs.

### 3.2 Out Of MVP

Do not implement these in MVP:

- Real-money gameplay.
- Payment or withdrawal.
- External wallet integration.
- Complex rank leagues.
- Tournament mode.
- Voice chat.
- Mobile native app.
- Full admin console.
- Full anti-cheat model.

Leave clean extension points for future releases.

## 4. Game Mode

MVP implements classic 3-player Dou Dizhu:

- One standard 54-card deck.
- Three players.
- Landlord gets 20 cards.
- Two farmers get 17 cards each.
- Three bottom cards.
- Basic multiplier model.
- Rocket and bomb multipliers.
- Game ends when one player has no cards.

Rule details are defined in [12-game-rules-engine.md](./12-game-rules-engine.md).

## 5. Core User Flows

### 5.1 Guest Quick Start

1. User opens web app.
2. User chooses guest login or auto guest entry.
3. Client calls login API.
4. User enters lobby.
5. User clicks quick start.
6. Server finds or creates a room.
7. Player enters room.
8. Game starts when three seats are filled and ready.

Acceptance:

- A first-time user can enter a playable match in under 3 clicks.
- If no real players are available, robots can fill empty seats after delay.

### 5.2 Room Game Flow

1. Three players join room.
2. Players ready.
3. Server shuffles and deals cards.
4. Bidding phase begins.
5. Landlord is selected.
6. Bottom cards are assigned to landlord.
7. Playing phase begins.
8. Current player plays cards or passes.
9. Server validates move.
10. Game ends when one hand is empty.
11. Settlement is broadcast.
12. Room returns to ready state or closes.

Acceptance:

- All state transitions are server-authoritative.
- Invalid client actions are rejected and logged.
- All clients receive identical public state.

### 5.3 Reconnect Flow

1. Player network disconnects.
2. Server marks player offline but keeps seat.
3. If turn reaches offline player, robot can temporarily act according to configured policy.
4. Player reconnects with token.
5. Server restores room state and private hand.
6. Player resumes control.

Acceptance:

- Reconnect works after page refresh.
- Player private cards are restored only to the rightful user.
- Spectator-like leakage of hidden cards must not occur.

## 6. Functional Requirements

### 6.1 Authentication

MVP authentication:

- Guest login.
- JWT access token.
- Refresh token optional for MVP, but architecture should not block it.

Registered account can be added later:

- Username/password.
- OAuth.
- Phone login.

### 6.2 Lobby

Lobby must support:

- Quick start.
- Room list query.
- Online player count.
- Current user's display name and avatar.

Room list data:

- Room ID.
- Mode.
- Current player count.
- Status.
- Base score.

### 6.3 Matchmaking

Quick start strategy:

1. Prefer room with waiting seats and same mode.
2. If not found, create room.
3. Fill with robots after configurable timeout.

### 6.4 Room

Room must support:

- Join.
- Leave before game starts.
- Ready.
- Kick robot only if real player enters before game starts.
- State broadcast.
- Chat emotes optional.

### 6.5 Gameplay

Server must handle:

- Shuffle.
- Deal.
- Bidding.
- Landlord selection.
- Bottom cards.
- Turn order.
- Card play validation.
- Pass validation.
- Timeout handling.
- Game over.
- Settlement.

### 6.6 Robot

Robot must support:

- Bidding decision.
- Play decision.
- Pass decision.
- Timeout replacement for disconnected users if policy permits.

Robot design is defined in [13-robot-ai-design.md](./13-robot-ai-design.md).

### 6.7 Records

Persist:

- Match metadata.
- Player snapshots.
- Initial hands.
- Bidding events.
- Play events.
- Settlement.
- Timestamps.

For security, records may contain full hands, but public APIs must not expose hidden cards to unauthorized users.

## 7. Non-Functional Requirements

### 7.1 Performance

MVP target:

- WebSocket action round-trip under 200 ms in local network.
- Rule validation under 5 ms per action.
- One server process supports at least 1,000 concurrent WebSocket connections in dev benchmark.
- Room tick operations should not block unrelated rooms.

### 7.2 Availability

MVP:

- Single backend instance acceptable.
- Redis recommended for room/session state in production.
- PostgreSQL or MySQL for persistence.

Production target:

- Stateless API nodes.
- WebSocket gateway nodes.
- Redis-backed session and room registry.
- Graceful shutdown.

### 7.3 Security

Requirements:

- Server authoritative state.
- Never trust client card lists, seat ownership, or turn identity.
- JWT validation for all protected APIs and WebSocket connections.
- Validate every action against server-side room state.
- Rate-limit login, room actions, and WebSocket messages.
- Avoid logging tokens.

### 7.4 Fairness

Requirements:

- Shuffle must use cryptographically secure randomness or a seedable audited RNG strategy.
- Persist shuffle seed or hash strategy for audit.
- Server owns all card dealing.
- Clients only receive legal private information.

### 7.5 Observability

Must log:

- Login.
- Room join/leave.
- Ready state.
- Game start.
- Bidding action.
- Play action.
- Invalid action.
- Disconnect/reconnect.
- Settlement.

Each log entry should include:

- trace_id.
- user_id.
- room_id if applicable.
- game_id if applicable.
- event_type.
- timestamp.

## 8. Success Metrics

MVP success:

- Three browser clients can complete a full match.
- One or more robots can replace missing players.
- Refreshing browser during a game restores state.
- Invalid card play is rejected.
- Match record can be replayed from persisted events.
- Unit tests cover core card type recognition and move comparison.

## 9. AI Development Policy

When using AI coding tools:

- Generate one bounded module per task.
- Use existing documentation as source of truth.
- Do not invent new gameplay rules without updating docs first.
- Keep generated code testable.
- Add tests with each game-rules change.
- Never expose hidden cards in API or WebSocket response.

Detailed AI coding rules are in [07-ai-coding-rules.md](./07-ai-coding-rules.md).

