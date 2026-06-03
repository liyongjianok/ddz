# Domain Model

## 1. Domain Goals

The domain model defines game concepts independently of transport, database, and UI.

The central rule:

`internal/game` must be pure and deterministic when provided the same inputs.

## 2. Aggregate Boundaries

### 2.1 User

Represents identity and profile. User is outside the game rules engine.

Fields:

- user_id.
- display_name.
- avatar_url.
- account_type.
- status.

### 2.2 Room

Room coordinates seats, readiness, active game, connections, and lifecycle.

Fields:

- room_id.
- mode.
- status.
- base_score.
- seats.
- current_game.
- created_at.
- updated_at.

Room owns:

- Seat assignment.
- Ready state.
- Active game instance.
- Reconnect state.

### 2.3 Game

Game is one Dou Dizhu match.

Fields:

- game_id.
- phase.
- deck.
- bottom_cards.
- players.
- landlord_seat_index.
- current_seat_index.
- last_play.
- pass_count.
- bidding_state.
- multiplier.
- event_seq.
- started_at.
- ended_at.

Game owns:

- Cards.
- Turn order.
- Bidding phase.
- Playing phase.
- Settlement.

## 3. Value Objects

### 3.1 Card

Card identity format:

```text
S3  spade 3
H3  heart 3
C3  club 3
D3  diamond 3
BJ  black joker
RJ  red joker
```

Suits:

- `S`: Spade.
- `H`: Heart.
- `C`: Club.
- `D`: Diamond.

Ranks:

- `3 4 5 6 7 8 9 T J Q K A 2 BJ RJ`

Rank order in Dou Dizhu:

```text
3 < 4 < 5 < 6 < 7 < 8 < 9 < T < J < Q < K < A < 2 < BJ < RJ
```

The suit has no comparison meaning in gameplay.

### 3.2 CardGroup

Represents a legal card pattern.

Fields:

- type.
- primary_rank.
- cards.
- length.
- attachments.

Group types:

- `single`
- `pair`
- `triple`
- `triple_with_single`
- `triple_with_pair`
- `straight`
- `pair_straight`
- `airplane`
- `airplane_with_singles`
- `airplane_with_pairs`
- `four_with_two_singles`
- `four_with_two_pairs`
- `bomb`
- `rocket`

### 3.3 Play

Fields:

- seat_index.
- user_id.
- cards.
- group.
- created_at.

### 3.4 Bid

Fields:

- seat_index.
- user_id.
- score.
- created_at.

Score:

- 0 means pass/no bid.
- 1, 2, 3 are bid scores.

## 4. Entities

### 4.1 PlayerState

Fields:

- user_id.
- seat_index.
- role.
- hand.
- status.
- is_robot.
- bid_score.
- remaining_count.

Status values:

- `joined`
- `ready`
- `playing`
- `offline`
- `left`

Role:

- `landlord`
- `farmer`
- null before assignment.

### 4.2 BiddingState

Fields:

- start_seat_index.
- current_seat_index.
- highest_bid.
- highest_bid_seat_index.
- bids.
- rounds.
- deadline_at.

Rules:

- Player can bid 0, 1, 2, or 3.
- A non-zero bid must be greater than current highest bid.
- Bid 3 immediately decides landlord.
- If no one bids, reshuffle/redeal or force strategy according to rules config.

MVP recommendation:

- If all players pass, redeal once.
- If repeated all-pass happens, randomly assign landlord with base score 1 to avoid endless loop.

### 4.3 PlayingState

Fields:

- current_seat_index.
- last_play.
- last_play_seat_index.
- pass_count.
- deadline_at.

Rules:

- If `last_play` is null, current player must play cards.
- If `last_play` belongs to current player because others passed, current player starts a new trick and may play any legal group.
- Player may pass only when responding to another player's active `last_play`.

## 5. Commands

Commands are intent from external systems to domain/application services.

```text
JoinRoom(user_id)
LeaveRoom(user_id)
Ready(user_id, ready)
StartGame()
PlaceBid(user_id, score)
PlayCards(user_id, cards)
Pass(user_id)
HandleTimeout(seat_index)
Disconnect(user_id)
Reconnect(user_id)
```

Command handling belongs in room/application service. Pure rule validation belongs in game package.

## 6. Events

Events are facts after successful state changes.

```text
RoomCreated
PlayerJoined
PlayerLeft
PlayerReadyChanged
GameStarted
CardsDealt
BidPlaced
LandlordDecided
BottomCardsRevealed
CardsPlayed
PlayerPassed
TurnChanged
TimeoutTriggered
PlayerDisconnected
PlayerReconnected
GameEnded
SettlementCompleted
```

Events should include:

- event_id.
- room_id.
- game_id if applicable.
- seq.
- actor_user_id if applicable.
- payload.
- created_at.

## 7. State Machine

### 7.1 Room State

```text
waiting -> playing -> settling -> waiting
waiting -> closed
playing -> closed only through forced admin shutdown or fatal abort
```

### 7.2 Game State

```text
created -> dealing -> bidding -> playing -> ended
created -> aborted
dealing -> aborted
bidding -> aborted
playing -> aborted
```

## 8. Invariants

Must always hold:

- A room has at most 3 seats.
- A game has exactly 3 players.
- A deck has exactly 54 unique cards.
- During active game, every card is in exactly one location:
  - player hand.
  - bottom cards before landlord assignment.
  - played event history.
- Current turn must point to an active seat.
- Only current player can bid or play.
- Player cannot play cards not in their hand.
- A play must be a legal card group.
- A response play must beat previous active play unless it is bomb/rocket according to rules.
- Hidden hands are private.

## 9. DTO Mapping

Domain objects should map to different DTOs:

### 9.1 PublicPlayerDTO

```json
{
  "user_id": "u_1",
  "display_name": "A",
  "seat_index": 0,
  "role": "farmer",
  "status": "online",
  "remaining_count": 17,
  "is_robot": false
}
```

### 9.2 PrivatePlayerDTO

```json
{
  "user_id": "u_1",
  "seat_index": 0,
  "hand": ["S3", "H3"]
}
```

Do not merge public and private DTOs in a way that risks leaking hidden cards.

## 10. Domain Acceptance Criteria

- Card parser rejects invalid cards.
- Deck generator creates 54 unique cards.
- Deal creates hands of 17, 17, 17 and bottom of 3.
- Card group recognizer identifies all MVP patterns.
- Move comparator follows Dou Dizhu ordering.
- Domain tests do not require database or WebSocket.

