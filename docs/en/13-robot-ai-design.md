# Robot AI Design

## 1. Purpose

Robot players allow:

- Single human to play immediately.
- Empty seats to be filled.
- Offline player auto-actions when policy allows.

Robot must always produce legal actions. Playing well is secondary to correctness.

## 2. Robot Types

### 2.1 Seat-Fill Robot

Used when room lacks human players.

Behavior:

- Joins waiting room after delay.
- Ready automatically.
- Plays full game.

### 2.2 Offline Proxy Robot

Used when a human disconnects during active game.

Behavior:

- Acts only on timeout.
- Does not immediately take over every turn unless configured.
- Human regains control on reconnect.

## 3. Design Principles

- Robot actions enter through the room command queue.
- Robot must use the same validation path as humans.
- Robot cannot mutate game state directly.
- Robot sees only information available to its seat plus public state.
- Robot strategy should be deterministic in tests with fixed seed.

## 4. Robot Input Snapshot

```go
type RobotSnapshot struct {
    RoomID string
    GameID string
    SeatIndex int
    Role Role
    Phase GamePhase
    Hand []Card
    PublicPlayers []PublicPlayerState
    BottomCards []Card
    CurrentSeatIndex int
    LastPlay *Play
    Multiplier int
    BidState BidStateView
}
```

Privacy:

- Robot receives own hand.
- Robot does not receive other players' hidden hands.

## 5. Robot Actions

```go
type RobotAction struct {
    Type RobotActionType
    BidScore int
    Cards []Card
}
```

Action types:

- `bid`
- `play`
- `pass`

## 6. Bidding Strategy

MVP heuristic:

Calculate hand strength:

- Rocket: +8.
- Each bomb: +6.
- Each 2: +2.
- Each joker: +3.
- Each A/K: +1.
- Long straight or airplane: +2.

Decision:

- Strength >= 16: bid 3 if legal.
- Strength >= 12: bid 2 if legal.
- Strength >= 8: bid 1 if legal.
- Otherwise pass.

If current highest bid is already higher:

- Only bid if desired score is greater than highest.
- Otherwise pass.

## 7. Playing Strategy

### 7.1 Opening A Trick

Priority:

1. If can finish all cards in one legal play, play them.
2. Play smallest single if hand is weak.
3. Prefer playing low combinations that reduce hand complexity:
   - straight.
   - pair straight.
   - airplane without attachments.
   - triple with attachment.
   - pair.
   - single.
4. Avoid bomb/rocket unless near winning.

### 7.2 Responding To Previous Play

Priority:

1. If a non-bomb legal response exists, play the smallest response that beats previous.
2. If opponent has very few cards, consider bomb.
3. If teammate/farmer partner played previous and landlord is not about to win, pass.
4. Otherwise pass if legal.

For MVP, teammate awareness can be simple:

- If robot is farmer and previous play belongs to another farmer, prefer pass.

### 7.3 Endgame Heuristic

If robot can win this turn:

- Play winning group.

If next opponent has 1 card:

- Prefer pair or larger group instead of single when opening.

If next opponent has 2 cards:

- Avoid playing pair if possible.

## 8. Legal Move Dependency

Robot must call:

```go
LegalResponses(hand, previous)
Recognize(cards)
CanBeat(candidate, previous)
```

Never hand-roll separate legality logic that can diverge from rules engine.

## 9. Timing

Robot action delay:

- Minimum: 500 ms.
- Default: 1,000 to 2,500 ms randomized.
- Timeout robot action: immediate after timeout command or within 300 ms.

Do not block room actor while sleeping.

Implementation:

- Schedule timer outside actor.
- Timer sends robot command into actor queue.

## 10. Difficulty Levels

MVP:

- `basic`

Future:

- `easy`: more random, weaker.
- `normal`: heuristic.
- `hard`: search-based.

## 11. Testing

Required tests:

- Robot bid is legal.
- Robot play is legal for random hands.
- Robot does not play cards outside hand.
- Robot can handle no legal response by passing.
- Robot opening move is legal.
- Robot with fixed seed is deterministic.

Property-style test recommendation:

- Generate random hands and previous plays.
- Ask robot for action.
- Validate action through rules engine.

## 12. Robot Acceptance Criteria

- Human can complete a game with two robots.
- Robot never causes invalid game state.
- Robot respects hidden information boundaries.
- Offline proxy does not prevent human reconnect.

