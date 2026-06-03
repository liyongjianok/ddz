package game

import (
	"errors"
	"math/rand"
	"testing"
)

func TestNewDeckHas54UniqueCards(t *testing.T) {
	deck := NewDeck()
	if len(deck) != DeckCardCount {
		t.Fatalf("len(deck) = %d, want %d", len(deck), DeckCardCount)
	}

	seen := make(map[string]struct{}, len(deck))
	for _, card := range deck {
		code := card.Code()
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate card %q found in deck", code)
		}
		seen[code] = struct{}{}
	}

	if _, ok := seen["BJ"]; !ok {
		t.Fatal("deck does not contain BJ")
	}
	if _, ok := seen["RJ"]; !ok {
		t.Fatal("deck does not contain RJ")
	}
}

func TestShuffleUsesInjectedRNG(t *testing.T) {
	deck := NewDeck()
	rngA := rand.New(rand.NewSource(42))
	rngB := rand.New(rand.NewSource(42))

	shuffledA := Shuffle(deck, rngA)
	shuffledB := Shuffle(deck, rngB)

	if len(shuffledA) != len(deck) || len(shuffledB) != len(deck) {
		t.Fatalf("shuffle changed deck length")
	}

	for i := range shuffledA {
		if shuffledA[i] != shuffledB[i] {
			t.Fatalf("shuffles with same seed differ at %d: %v vs %v", i, shuffledA[i], shuffledB[i])
		}
	}

	for i := range deck {
		if deck[i] != NewDeck()[i] {
			t.Fatal("Shuffle mutated original deck")
		}
	}
}

func TestDealReturnsExpectedCounts(t *testing.T) {
	deck := Shuffle(NewDeck(), rand.New(rand.NewSource(7)))

	hands, bottom, err := Deal(deck)
	if err != nil {
		t.Fatalf("Deal() error = %v", err)
	}

	for i := range hands {
		if len(hands[i]) != HandCardCount {
			t.Fatalf("len(hands[%d]) = %d, want %d", i, len(hands[i]), HandCardCount)
		}
	}
	if len(bottom) != BottomCardCount {
		t.Fatalf("len(bottom) = %d, want %d", len(bottom), BottomCardCount)
	}

	seen := make(map[string]struct{}, DeckCardCount)
	for _, hand := range hands {
		for _, card := range hand {
			code := card.Code()
			if _, exists := seen[code]; exists {
				t.Fatalf("duplicate dealt card %q", code)
			}
			seen[code] = struct{}{}
		}
	}
	for _, card := range bottom {
		code := card.Code()
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate dealt bottom card %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) != DeckCardCount {
		t.Fatalf("dealt unique card count = %d, want %d", len(seen), DeckCardCount)
	}
}

func TestDealRejectsInvalidDeck(t *testing.T) {
	shortDeck := NewDeck()[:10]
	if _, _, err := Deal(shortDeck); !errors.Is(err, ErrInvalidDeck) {
		t.Fatalf("Deal(shortDeck) error = %v, want ErrInvalidDeck", err)
	}

	deck := NewDeck()
	deck[1] = deck[0]
	if _, _, err := Deal(deck); !errors.Is(err, ErrInvalidDeck) {
		t.Fatalf("Deal(duplicateDeck) error = %v, want ErrInvalidDeck", err)
	}
}
