# Test Cases

## 1. Test Strategy

Testing must focus on correctness of game rules, server authority, hidden-information safety, and reconnect behavior.

Priority order:

1. Rules engine unit tests.
2. Game state machine tests.
3. Room actor tests.
4. WebSocket protocol tests.
5. API integration tests.
6. Frontend interaction tests.
7. End-to-end smoke tests.

## 2. Rules Engine Unit Tests

### TC-RULE-001 Card Parse Valid

Input:

- `S3`, `HT`, `DA`, `C2`, `BJ`, `RJ`.

Expected:

- All parse successfully.
- Rank and suit are correct.

### TC-RULE-002 Card Parse Invalid

Input:

- `X3`, `S1`, `B`, `RJ1`, empty string.

Expected:

- Parser returns error.

### TC-RULE-003 Deck Unique

Expected:

- Deck length is 54.
- No duplicate card code.
- Contains both jokers.

### TC-RULE-004 Deal Counts

Expected:

- Three hands of 17.
- Bottom cards length 3.
- Total unique cards 54.

### TC-RULE-005 Recognize Single

Input:

- `S3`

Expected:

- type `single`, rank `3`.

### TC-RULE-006 Recognize Pair

Input:

- `S3`, `H3`

Expected:

- type `pair`, rank `3`.

### TC-RULE-007 Recognize Triple

Input:

- `S3`, `H3`, `D3`

Expected:

- type `triple`, rank `3`.

### TC-RULE-008 Recognize Triple With Single

Input:

- `S3`, `H3`, `D3`, `S4`

Expected:

- type `triple_with_single`, primary rank `3`.

### TC-RULE-009 Recognize Triple With Pair

Input:

- `S3`, `H3`, `D3`, `S4`, `H4`

Expected:

- type `triple_with_pair`, primary rank `3`.

### TC-RULE-010 Recognize Straight

Input:

- `S3`, `H4`, `D5`, `S6`, `C7`

Expected:

- type `straight`, primary rank `7`, length `5`.

### TC-RULE-011 Straight Cannot Include 2 Or Jokers

Input:

- `ST`, `HJ`, `DQ`, `SK`, `CA`, `D2`

Expected:

- invalid group.

### TC-RULE-012 Recognize Pair Straight

Input:

- `S3`, `H3`, `S4`, `H4`, `S5`, `H5`

Expected:

- type `pair_straight`, primary rank `5`, length `3`.

### TC-RULE-013 Recognize Airplane

Input:

- `S3`, `H3`, `D3`, `S4`, `H4`, `D4`

Expected:

- type `airplane`, primary rank `4`, length `2`.

### TC-RULE-014 Recognize Bomb

Input:

- `S7`, `H7`, `D7`, `C7`

Expected:

- type `bomb`, rank `7`.

### TC-RULE-015 Recognize Rocket

Input:

- `BJ`, `RJ`

Expected:

- type `rocket`.

### TC-RULE-016 Compare Same Type

Input:

- single 4 vs single 3.

Expected:

- single 4 beats single 3.

### TC-RULE-017 Compare Different Non-Bomb

Input:

- pair 4 vs single 3.

Expected:

- cannot compare / does not beat.

### TC-RULE-018 Bomb Beats Non-Bomb

Expected:

- bomb 3 beats single A.

### TC-RULE-019 Higher Bomb Beats Lower Bomb

Expected:

- bomb 8 beats bomb 7.

### TC-RULE-020 Rocket Beats Bomb

Expected:

- rocket beats any bomb.

## 3. Game State Tests

### TC-GAME-001 Initialize Game

Expected:

- Three players.
- Phase is bidding.
- Hands and bottom cards assigned.
- Current seat is valid.

### TC-GAME-002 Bid Out Of Turn

Expected:

- Error `not_player_turn`.

### TC-GAME-003 Invalid Bid Lower Than Highest

Expected:

- Error `invalid_bid`.

### TC-GAME-004 Bid 3 Decides Landlord

Expected:

- Phase changes to playing.
- Landlord has 20 cards.
- Farmers have 17 cards.
- Current turn is landlord.
- Multiplier includes bid score.

### TC-GAME-005 All Pass Handling

Expected:

- Follows configured all-pass policy.
- No infinite loop.

### TC-GAME-006 Play Out Of Turn

Expected:

- Error `not_player_turn`.

### TC-GAME-007 Play Missing Card

Expected:

- Error `invalid_card_set`.

### TC-GAME-008 Opening Cannot Pass

Expected:

- Error `cannot_pass`.

### TC-GAME-009 Valid Play Removes Cards

Expected:

- Cards removed from hand.
- Last play updated.
- Turn advances.

### TC-GAME-010 Response Must Beat Last Play

Expected:

- Lower same pattern rejected.
- Valid higher same pattern accepted.

### TC-GAME-011 Pass Clears Trick After Two Passes

Expected:

- If two opponents pass, last player starts new trick.

### TC-GAME-012 Game Ends On Empty Hand

Expected:

- Phase becomes ended.
- Winner side computed.

## 4. Room Actor Tests

### TC-ROOM-001 Join Three Players

Expected:

- Seats 0, 1, 2 assigned.
- Fourth player rejected.

### TC-ROOM-002 User Cannot Join Twice

Expected:

- Same user receives existing seat or error.

### TC-ROOM-003 Ready Starts Game

Expected:

- When all seats ready, game starts.

### TC-ROOM-004 Snapshot Privacy

Expected:

- Snapshot for player A includes A hand only.
- Player B/C hands are hidden.

### TC-ROOM-005 Serialized Commands

Expected:

- Concurrent ready commands produce one game start.

### TC-ROOM-006 Timeout Command

Expected:

- Timeout action is legal.
- No timeout acts after turn already changed.

## 5. WebSocket Tests

### TC-WS-001 Connect Without Token

Expected:

- Connection rejected.

### TC-WS-002 Connect With Invalid Room

Expected:

- Connection rejected or error with `room_not_found`.

### TC-WS-003 Connect Sends Snapshot

Expected:

- First valid connection receives `room.snapshot`.

### TC-WS-004 Unknown Message Type

Expected:

- Server sends `error` with code `bad_request`.

### TC-WS-005 Ready Message

Expected:

- Ready event broadcast.

### TC-WS-006 Invalid Play Message

Expected:

- Server sends `error`.
- State unchanged.

### TC-WS-007 Private Hand Update

Expected:

- Actor receives updated own hand.
- Other players receive only remaining count.

## 6. API Tests

### TC-API-001 Guest Login

Expected:

- Returns user and access token.

### TC-API-002 Auth Me

Expected:

- Valid token returns profile.
- Invalid token rejected.

### TC-API-003 Quick Start

Expected:

- Returns room and WebSocket URL.

### TC-API-004 Room List Privacy

Expected:

- No card data in response.

### TC-API-005 My Records

Expected:

- Returns only current user's records.

## 7. Reconnect Tests

### TC-REC-001 Disconnect Marks Offline

Expected:

- Player status becomes offline.
- Other players receive event.

### TC-REC-002 Reconnect Restores Hand

Expected:

- Reconnected player receives own current hand.

### TC-REC-003 Reconnect Does Not Leak Others

Expected:

- Reconnect snapshot hides other hands.

### TC-REC-004 Timeout While Offline

Expected:

- Auto action occurs if policy allows.
- Player can reconnect afterward.

## 8. Robot Tests

### TC-BOT-001 Robot Bid Is Legal

Expected:

- Robot emits only allowed bid score.

### TC-BOT-002 Robot Play Is Legal

Expected:

- Robot never plays cards not in hand.
- Robot play beats previous when required.

### TC-BOT-003 Robot Fills Seats

Expected:

- Empty waiting room fills after configured delay.

## 9. Frontend Tests

### TC-FE-001 Login Page

Expected:

- Guest login stores token and navigates to lobby.

### TC-FE-002 Lobby Quick Start

Expected:

- Quick start opens room page.

### TC-FE-003 Room Snapshot Render

Expected:

- Seats, hand, phase, turn, and countdown render.

### TC-FE-004 Card Selection

Expected:

- Clicking card toggles selected state.

### TC-FE-005 Action Buttons

Expected:

- Play/pass disabled when not user's turn.
- Ready button shown before game starts.

### TC-FE-006 Reconnect UI

Expected:

- Shows reconnecting state.
- Applies new snapshot after reconnect.

## 10. End-To-End Smoke Tests

### TC-E2E-001 Three Human Clients Finish Game

Expected:

- Three clients login.
- Join same room.
- Ready.
- Complete a full game.
- Settlement displayed.

### TC-E2E-002 One Human Two Robots Finish Game

Expected:

- Human quick starts.
- Robots fill.
- Game completes.

### TC-E2E-003 Refresh During Game

Expected:

- Browser refresh reconnects and resumes.

