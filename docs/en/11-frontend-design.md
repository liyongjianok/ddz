# Frontend Design

## 1. Design Goal

Create a web Dou Dizhu interface that feels like a polished game table, but remains clear, fast, and practical for browser play.

The UI must prioritize:

- Readability of cards.
- Clear current turn.
- Fast action controls.
- Reconnect clarity.
- No hidden-information leaks.

## 2. Pages

### 2.1 Login Page

Purpose:

- Let user enter quickly as guest.

Elements:

- Display name input.
- Guest login button.
- Minimal product title.

Acceptance:

- User can login without registration.

### 2.2 Lobby Page

Purpose:

- Show player profile and room entry options.

Elements:

- Header with display name, coin balance, connection state.
- Quick start primary button.
- Room list.
- Mode/base score selector.
- Online summary.

Acceptance:

- Quick start enters a room.
- Room list is scannable.

### 2.3 Room Page

Purpose:

- Main gameplay screen.

Elements:

- Three player seats.
- Center play area.
- Bottom cards area.
- Current trick display.
- User hand.
- Action controls.
- Countdown timer.
- Connection/reconnect indicator.
- Settlement overlay.

Acceptance:

- User can complete a full game.

## 3. Room Layout

Desktop layout:

```text
------------------------------------------------
| Top bar: room id, phase, multiplier, timer    |
------------------------------------------------
|          Opponent seat / played cards         |
|                                                |
| Left seat        Center table        Right seat|
|                                                |
|          Own played cards / action prompt     |
------------------------------------------------
| Own hand cards                                 |
| Action buttons: Ready / Bid / Play / Pass     |
------------------------------------------------
```

Mobile layout:

- Keep own hand at bottom.
- Stack opponent seats near top.
- Use compact controls.
- Avoid requiring horizontal scroll except for hand cards.

## 4. Visual Style

Recommended:

- Table background: deep green or muted teal.
- Cards: white/off-white with clear suit colors.
- Red suits: red.
- Black suits: near black.
- Action buttons: high contrast.
- Current turn: strong outline or glow.
- Disabled controls: visibly muted.

Avoid:

- Overly decorative landing-page layout.
- Tiny card text.
- Hidden buttons behind hover-only UI.
- Heavy animation that delays play.

## 5. Components

### 5.1 Card

Props:

```ts
type CardProps = {
  code: string;
  selected?: boolean;
  disabled?: boolean;
  faceDown?: boolean;
  onClick?: () => void;
};
```

Behavior:

- Click toggles selection when enabled.
- Selected card moves upward slightly.
- Face-down card hides code.

### 5.2 Hand

Props:

```ts
type HandProps = {
  cards: CardCode[];
  selected: CardCode[];
  sortMode: "rank" | "suit";
  onToggle: (card: CardCode) => void;
};
```

Behavior:

- Cards overlap slightly to fit screen.
- Selected order should preserve hand order.

### 5.3 PlayerSeat

Props:

```ts
type PlayerSeatProps = {
  player: PublicPlayer;
  isCurrentTurn: boolean;
  lastPlayedCards: CardCode[];
  countdown?: number;
};
```

Display:

- Avatar.
- Name.
- Role.
- Remaining count.
- Online/offline.
- Robot badge if applicable.

### 5.4 ActionPanel

Props:

```ts
type ActionPanelProps = {
  phase: GamePhase;
  isMyTurn: boolean;
  canPass: boolean;
  selectedCards: CardCode[];
  onReady: () => void;
  onBid: (score: 0 | 1 | 2 | 3) => void;
  onPlay: () => void;
  onPass: () => void;
  onHint: () => void;
};
```

Rules:

- Show only phase-relevant actions.
- Disable actions when not player's turn.
- Play button disabled if no selected cards.

### 5.5 SettlementModal

Displays:

- Winner side.
- Score deltas.
- Multiplier details.
- Continue/ready button.

## 6. Frontend State Model

### 6.1 Auth Store

Fields:

- user.
- token.
- status.

Actions:

- guestLogin.
- loadMe.
- logout.

### 6.2 Lobby Store

Fields:

- summary.
- rooms.
- loading.

Actions:

- fetchSummary.
- fetchRooms.
- quickStart.

### 6.3 Room Store

Fields:

- room.
- players.
- game.
- me.
- selectedCards.
- connectionState.
- lastError.

Actions:

- connect.
- disconnect.
- applySnapshot.
- applyEvent.
- toggleCard.
- clearSelection.
- sendReady.
- sendBid.
- sendPlay.
- sendPass.

## 7. WebSocket Client Behavior

States:

- `idle`
- `connecting`
- `connected`
- `reconnecting`
- `disconnected`
- `failed`

Behavior:

- Connect when room page opens.
- Send heartbeat every 15 seconds.
- On close, reconnect with exponential backoff.
- After reconnect, wait for authoritative snapshot.
- Do not reuse stale local game state as truth.

## 8. Card Sorting

Default sort:

```text
RJ, BJ, 2, A, K, Q, J, T, 9 ... 3
```

Alternative:

- Group by rank count for easier pattern selection.

Client sorting affects only display, not server logic.

## 9. Error Feedback

Examples:

- `not_player_turn`: "还没轮到你出牌"
- `invalid_card_set`: "这组牌型不合法"
- `cannot_pass`: "当前不能不出"
- `connection_lost`: "连接断开，正在重连"

Do not display raw backend stack traces.

## 10. Accessibility

Minimum:

- Buttons have text labels or accessible labels.
- Card elements have readable labels.
- Color is not the only turn indicator.
- Keyboard selection optional but recommended.

## 11. Frontend Acceptance Criteria

- Guest login works.
- Lobby quick start works.
- Room renders server snapshot.
- Player can ready, bid, play, pass.
- Cards are readable on desktop and mobile.
- Reconnect state is visible.
- Hidden cards for other players are never present in frontend state.

