package game

import (
	"errors"
	"testing"
)

func TestParseCardValid(t *testing.T) {
	testCases := []struct {
		code string
		card Card
	}{
		{code: "S3", card: Card{Suit: SuitSpade, Rank: Rank3}},
		{code: "HT", card: Card{Suit: SuitHeart, Rank: RankT}},
		{code: "DA", card: Card{Suit: SuitDiamond, Rank: RankA}},
		{code: "C2", card: Card{Suit: SuitClub, Rank: Rank2}},
		{code: "BJ", card: Card{Suit: SuitNone, Rank: RankBlackJoker}},
		{code: "RJ", card: Card{Suit: SuitNone, Rank: RankRedJoker}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.code, func(t *testing.T) {
			card, err := ParseCard(tc.code)
			if err != nil {
				t.Fatalf("ParseCard(%q) error = %v", tc.code, err)
			}
			if card != tc.card {
				t.Fatalf("ParseCard(%q) = %#v, want %#v", tc.code, card, tc.card)
			}
			if got := card.Code(); got != tc.code {
				t.Fatalf("Code() = %q, want %q", got, tc.code)
			}
		})
	}
}

func TestParseCardInvalid(t *testing.T) {
	testCases := []string{
		"",
		"B",
		"X3",
		"S1",
		"RJ1",
		"SBJ",
	}

	for _, code := range testCases {
		code := code
		t.Run(code, func(t *testing.T) {
			_, err := ParseCard(code)
			if !errors.Is(err, ErrInvalidCardCode) {
				t.Fatalf("ParseCard(%q) error = %v, want ErrInvalidCardCode", code, err)
			}
		})
	}
}
