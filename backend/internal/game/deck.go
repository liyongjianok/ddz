package game

import (
	"errors"
	"fmt"
)

const (
	PlayerCount     = 3
	HandCardCount   = 17
	BottomCardCount = 3
	DeckCardCount   = PlayerCount*HandCardCount + BottomCardCount
)

var ErrInvalidDeck = errors.New("invalid deck")

type RNG interface {
	Intn(n int) int
}

func NewDeck() []Card {
	deck := make([]Card, 0, DeckCardCount)
	for _, rank := range standardRanks {
		for _, suit := range standardSuits {
			deck = append(deck, Card{
				Suit: suit,
				Rank: rank,
			})
		}
	}

	deck = append(deck,
		Card{Suit: SuitNone, Rank: RankBlackJoker},
		Card{Suit: SuitNone, Rank: RankRedJoker},
	)

	return deck
}

func Shuffle(deck []Card, rng RNG) []Card {
	shuffled := append([]Card(nil), deck...)
	if rng == nil {
		return shuffled
	}

	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func Deal(deck []Card) (hands [PlayerCount][]Card, bottom []Card, err error) {
	if err := validateDeck(deck); err != nil {
		return hands, nil, err
	}

	for i := 0; i < PlayerCount; i++ {
		start := i * HandCardCount
		end := start + HandCardCount
		hands[i] = append([]Card(nil), deck[start:end]...)
	}

	bottomStart := PlayerCount * HandCardCount
	bottom = append([]Card(nil), deck[bottomStart:]...)
	return hands, bottom, nil
}

func validateDeck(deck []Card) error {
	if len(deck) != DeckCardCount {
		return fmt.Errorf("%w: expected %d cards, got %d", ErrInvalidDeck, DeckCardCount, len(deck))
	}

	seen := make(map[string]struct{}, len(deck))
	for _, card := range deck {
		code := card.Code()
		if _, exists := seen[code]; exists {
			return fmt.Errorf("%w: duplicate card %s", ErrInvalidDeck, code)
		}
		seen[code] = struct{}{}
	}

	return nil
}
