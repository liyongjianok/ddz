# AI Coding Rules

## 1. Purpose

This document is the operating manual for ChatGPT, Claude Code, Codex, or any AI agent generating this project.

The goal is controlled incremental development. Do not generate the whole project in one pass.

## 2. Golden Rules

1. Read the relevant docs before coding.
2. Implement one bounded module per task.
3. Keep game rules server-authoritative.
4. Never expose hidden cards to unauthorized clients.
5. Write tests for rules and state transitions.
6. Prefer simple, explicit code over clever abstractions.
7. Preserve package boundaries from architecture docs.
8. Do not invent APIs or protocol fields without updating docs.
9. Do not add real-money semantics.
10. Do not silently change Dou Dizhu rules.

## 3. Required Workflow For Every Module

For each coding task:

1. Identify target module.
2. Read related docs.
3. List inputs and outputs.
4. Implement code.
5. Add tests.
6. Run tests.
7. Summarize files changed and verification.

## 4. Backend Coding Standards

### 4.1 Go Style

- Use Go 1.22+.
- Use idiomatic package names.
- Avoid global mutable state except configuration constants.
- Prefer dependency injection through constructors.
- Return typed errors where useful.
- Keep functions small enough to test.
- Use `context.Context` for IO and long-running operations.
- Use `time.Time` in UTC for persisted times.

### 4.2 Package Dependency Rules

Allowed:

- `room` depends on `game`.
- `ws` depends on `room`.
- `api handlers` depend on services.
- `robot` depends on `game` snapshots and strategy interfaces.

Forbidden:

- `game` depending on `ws`, `http`, `db`, or `redis`.
- `game` reading environment variables.
- `game` logging transport details.
- frontend DTOs imported into backend domain.

### 4.3 Error Handling

Use stable codes:

```text
unauthorized
room_not_found
not_player_turn
invalid_bid
invalid_card_set
cannot_pass
state_conflict
internal_error
```

Do not expose stack traces to clients.

### 4.4 Logging

Use structured logging:

- event_type.
- trace_id.
- user_id.
- room_id.
- game_id.
- request_id.

Never log:

- JWT tokens.
- password hashes.
- full hidden hands in normal application logs.

## 5. Frontend Coding Standards

### 5.1 TypeScript

- Use strict TypeScript.
- Define protocol types in one place.
- Avoid `any` unless the boundary is truly dynamic.
- Keep API and WebSocket clients separated from UI components.

### 5.2 UI State

State categories:

- Auth state.
- Lobby state.
- Room snapshot state.
- Local card selection state.
- Connection state.

Do not store hidden cards for other players.

### 5.3 UX Rules

- Always show current phase.
- Always show current turn.
- Always show countdown.
- Disable illegal action buttons.
- Show server error feedback.
- On reconnect, show syncing state until snapshot arrives.

## 6. Game Rules Coding Rules

### 6.1 Determinism

Rules functions should be deterministic:

```go
Recognize(cards []Card) (CardGroup, error)
CanBeat(candidate CardGroup, previous CardGroup) bool
LegalMoves(hand []Card, previous *CardGroup) []CardGroup
```

### 6.2 Test Matrix

Every rules implementation must test:

- Single.
- Pair.
- Triple.
- Triple with single.
- Triple with pair.
- Straight.
- Pair straight.
- Airplane.
- Bomb.
- Rocket.
- Invalid groups.
- Bomb beats non-bomb.
- Rocket beats bomb.
- Higher same pattern beats lower same pattern.
- Different non-bomb patterns cannot compare.

## 7. WebSocket Coding Rules

- Validate token on upgrade.
- Bind one connection to one user and room.
- All incoming messages must pass schema validation.
- Unknown message type returns error.
- Use request IDs for response correlation.
- Send player-specific snapshots.
- Do not broadcast private hand data.
- Handle disconnect and reconnect.
- Rate-limit action messages.

## 8. Room Actor Coding Rules

- One serialized command path per room.
- No concurrent mutation of room state.
- All room state changes emit events.
- Timer events enter through the same command queue.
- Robot actions enter through the same command queue.
- Room manager owns room lifecycle.

## 9. Database Coding Rules

- Use migrations.
- Use transactions for game completion and settlement.
- Keep event log append-only.
- Do not update historical game events.
- Store sensitive full hands only in game records, not public room APIs.

## 10. Testing Requirements

### 10.1 Unit Tests

Required:

- Card parser.
- Deck generator.
- Shuffle/deal.
- Card group recognition.
- Move comparison.
- Bidding rules.
- Turn transition.
- Settlement.

### 10.2 Integration Tests

Required:

- Guest login.
- Quick start.
- WebSocket connect.
- Ready starts game.
- Full game with scripted moves if feasible.
- Reconnect snapshot.

### 10.3 Frontend Tests

Recommended:

- Card rendering.
- Selection behavior.
- Action button states.
- WebSocket event reducer.
- Reconnect UI state.

## 11. Prompt Template For AI Agents

Use this template for each module:

```text
You are implementing module: <module name>.
Read these docs first:
- docs/<doc1>
- docs/<doc2>

Constraints:
- Follow docs/07-ai-coding-rules.md.
- Do not implement unrelated modules.
- Add tests.
- Do not expose hidden cards.

Deliverables:
- Code files.
- Tests.
- Brief verification summary.
```

## 12. Module Order For AI Generation

Recommended:

1. Go project scaffold.
2. `internal/game` card/deck.
3. `internal/game` card group recognizer.
4. `internal/game` comparator.
5. `internal/game` state machine.
6. Room actor in memory.
7. HTTP auth and lobby.
8. WebSocket protocol.
9. Frontend scaffold.
10. Frontend room UI.
11. Robot strategy.
12. Reconnect.
13. Persistence.
14. Deployment.

## 13. Anti-Patterns

Avoid:

- One giant `main.go`.
- WebSocket handler directly mutating card hands.
- Rule logic duplicated in frontend and backend as source of truth.
- Storing cards as ambiguous display strings only.
- Broadcasting complete game state to all clients.
- Using sleeps for deterministic tests.
- Random behavior in tests without fixed seed.
- Real-money balance naming like cash, deposit, withdraw.

## 14. Acceptance Criteria For AI Output

Each AI-generated module is acceptable only if:

- It compiles.
- It has focused tests.
- It follows documented package boundaries.
- It does not introduce hidden card leakage.
- It includes a short summary of implementation and tests.

