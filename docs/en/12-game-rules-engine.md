# Game Rules Engine

## 1. Purpose

This document defines the Dou Dizhu rules implemented by the backend rules engine.

The rules engine must be:

- Deterministic.
- Unit-testable.
- Independent from HTTP/WebSocket/database.
- The single source of truth for gameplay validation.

## 2. Deck

Use one standard 54-card deck:

- 52 suited cards.
- Black joker.
- Red joker.

Card codes:

```text
S3 H3 C3 D3 ... SA HA CA DA S2 H2 C2 D2 BJ RJ
```

Rank order:

```text
3 < 4 < 5 < 6 < 7 < 8 < 9 < T < J < Q < K < A < 2 < BJ < RJ
```

Suits do not affect comparison.

## 3. Deal

For three players:

- Player 0: 17 cards.
- Player 1: 17 cards.
- Player 2: 17 cards.
- Bottom cards: 3 cards.

Server shuffles and deals.

## 4. Bidding Rules

MVP uses score bidding:

- Legal bid values: 0, 1, 2, 3.
- 0 means pass.
- Non-zero bid must be higher than current highest bid.
- Bid 3 immediately selects landlord.
- If bidding completes without 3, highest bidder becomes landlord.
- If all players pass, use all-pass policy.

All-pass policy:

1. Redeal once.
2. If all players pass again, randomly assign landlord and set bid score to 1.

After landlord selected:

- Landlord receives bottom cards.
- Landlord starts first play.
- Bid score contributes to multiplier.

## 5. Card Group Types

### 5.1 Single

One card.

Example:

```text
S3
```

### 5.2 Pair

Two cards of same rank.

Example:

```text
S3 H3
```

Jokers cannot form normal pair.

### 5.3 Triple

Three cards of same rank.

Example:

```text
S3 H3 D3
```

### 5.4 Triple With Single

Three cards of same rank plus one single attachment.

Example:

```text
S3 H3 D3 S4
```

### 5.5 Triple With Pair

Three cards of same rank plus one pair attachment.

Example:

```text
S3 H3 D3 S4 H4
```

### 5.6 Straight

Five or more consecutive single ranks.

Rules:

- Minimum length 5.
- Cannot include 2.
- Cannot include jokers.

Example:

```text
S3 H4 D5 C6 S7
```

### 5.7 Pair Straight

Three or more consecutive pairs.

Rules:

- Minimum 3 pairs.
- Cannot include 2.
- Cannot include jokers.

Example:

```text
S3 H3 S4 H4 S5 H5
```

### 5.8 Airplane

Two or more consecutive triples without attachments.

Rules:

- Minimum 2 triples.
- Cannot include 2.
- Cannot include jokers.

Example:

```text
S3 H3 D3 S4 H4 D4
```

### 5.9 Airplane With Singles

Consecutive triples plus same number of single attachments.

Example:

```text
333444 + 5 6
```

For two triples, exactly two single attachments.

### 5.10 Airplane With Pairs

Consecutive triples plus same number of pair attachments.

Example:

```text
333444 + 55 66
```

For two triples, exactly two pair attachments.

### 5.11 Four With Two Singles

Four cards of same rank plus two single attachments.

Example:

```text
3333 + 4 5
```

MVP permits this pattern.

### 5.12 Four With Two Pairs

Four cards of same rank plus two pair attachments.

Example:

```text
3333 + 44 55
```

MVP permits this pattern.

### 5.13 Bomb

Four cards of same rank.

Example:

```text
S7 H7 D7 C7
```

### 5.14 Rocket

Both jokers.

Example:

```text
BJ RJ
```

Rocket is the highest group.

## 6. Comparison Rules

### 6.1 Same Type

For non-bomb, non-rocket groups:

- Candidate must have same type and same structure length.
- Candidate primary rank must be higher than previous primary rank.

Examples:

- single 4 beats single 3.
- pair 6 beats pair 5.
- straight 4-8 beats straight 3-7.
- straight 4-9 does not compare with straight 3-7 because length differs.

### 6.2 Bomb

- Bomb beats any non-bomb and non-rocket group.
- Higher bomb beats lower bomb.
- Bomb cannot beat rocket.

### 6.3 Rocket

- Rocket beats everything.
- Nothing beats rocket.

### 6.4 Different Type

Different non-bomb types cannot compare.

## 7. Turn Rules

- Landlord starts the first play.
- Turn order follows seat index clockwise: 0 -> 1 -> 2 -> 0.
- Player must play cards when opening a trick.
- Player may pass only when responding to another player's active play.
- After two consecutive passes, the last player who played starts a new trick.

## 8. Timeout Rules

Bidding timeout:

- Default action: pass if legal.
- If forced action needed, choose lowest legal bid.

Playing timeout:

- If pass is legal, auto pass.
- If pass is not legal, play the smallest legal group according to robot strategy.

## 9. Multiplier Rules

MVP multiplier:

- Initial multiplier: bid score, minimum 1.
- Each bomb doubles multiplier.
- Rocket doubles multiplier.

Optional future multipliers:

- Spring.
- Anti-spring.
- No shuffle.
- Double score room.

MVP may skip spring/anti-spring unless explicitly requested later.

## 10. Settlement

If landlord wins:

- Landlord gains `base_score * multiplier * 2`.
- Each farmer loses `base_score * multiplier`.

If farmers win:

- Landlord loses `base_score * multiplier * 2`.
- Each farmer gains `base_score * multiplier`.

The sum of deltas must be zero.

## 11. Function Contracts

Recommended Go contracts:

```go
func ParseCard(code string) (Card, error)
func NewDeck() []Card
func Shuffle(deck []Card, rng RNG) []Card
func Deal(deck []Card) (hands [3][]Card, bottom []Card, err error)
func Recognize(cards []Card) (CardGroup, error)
func CanBeat(candidate CardGroup, previous CardGroup) bool
func RemoveCards(hand []Card, cards []Card) ([]Card, error)
func ContainsCards(hand []Card, cards []Card) bool
func LegalResponses(hand []Card, previous *CardGroup) []CardGroup
```

## 12. Edge Cases

Must handle:

- Duplicate card in selected play.
- Card not in hand.
- Empty play.
- Invalid group.
- Straight containing 2.
- Straight containing joker.
- Airplane attachment ambiguity.
- Four-with-two not confused with bomb.
- Rocket only exactly two jokers.

## 13. Rules Acceptance Criteria

- All group types recognized.
- Invalid groups rejected.
- Comparison follows rules.
- Settlement sum is zero.
- Tests cover edge cases.

