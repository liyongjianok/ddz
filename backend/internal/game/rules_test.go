package game

import (
	"errors"
	"reflect"
	"testing"
)

func TestRecognizeValidGroups(t *testing.T) {
	testCases := []struct {
		name        string
		codes       []string
		wantType    CardGroupType
		wantPrimary Rank
		wantLength  int
		wantAttach  int
	}{
		{name: "single", codes: []string{"S3"}, wantType: CardGroupTypeSingle, wantPrimary: Rank3, wantLength: 1},
		{name: "pair", codes: []string{"S3", "H3"}, wantType: CardGroupTypePair, wantPrimary: Rank3, wantLength: 2},
		{name: "triple", codes: []string{"S3", "H3", "D3"}, wantType: CardGroupTypeTriple, wantPrimary: Rank3, wantLength: 3},
		{name: "triple_with_single", codes: []string{"S3", "H3", "D3", "S4"}, wantType: CardGroupTypeTripleWithSingle, wantPrimary: Rank3, wantLength: 4, wantAttach: 1},
		{name: "triple_with_pair", codes: []string{"S3", "H3", "D3", "S4", "H4"}, wantType: CardGroupTypeTripleWithPair, wantPrimary: Rank3, wantLength: 5, wantAttach: 2},
		{name: "straight", codes: []string{"S3", "H4", "D5", "S6", "C7"}, wantType: CardGroupTypeStraight, wantPrimary: Rank7, wantLength: 5},
		{name: "pair_straight", codes: []string{"S3", "H3", "S4", "H4", "S5", "H5"}, wantType: CardGroupTypePairStraight, wantPrimary: Rank5, wantLength: 3},
		{name: "airplane", codes: []string{"S3", "H3", "D3", "S4", "H4", "D4"}, wantType: CardGroupTypeAirplane, wantPrimary: Rank4, wantLength: 2},
		{name: "airplane_with_singles", codes: []string{"S3", "H3", "D3", "S4", "H4", "D4", "S5", "H6"}, wantType: CardGroupTypeAirplaneWithSingles, wantPrimary: Rank4, wantLength: 2, wantAttach: 2},
		{name: "airplane_with_pairs", codes: []string{"S3", "H3", "D3", "S4", "H4", "D4", "S5", "H5", "S6", "H6"}, wantType: CardGroupTypeAirplaneWithPairs, wantPrimary: Rank4, wantLength: 2, wantAttach: 4},
		{name: "four_with_two_singles", codes: []string{"S7", "H7", "D7", "C7", "S4", "H5"}, wantType: CardGroupTypeFourWithTwoSingles, wantPrimary: Rank7, wantLength: 6, wantAttach: 2},
		{name: "four_with_two_pairs", codes: []string{"S7", "H7", "D7", "C7", "S4", "H4", "S5", "H5"}, wantType: CardGroupTypeFourWithTwoPairs, wantPrimary: Rank7, wantLength: 8, wantAttach: 4},
		{name: "bomb", codes: []string{"S7", "H7", "D7", "C7"}, wantType: CardGroupTypeBomb, wantPrimary: Rank7, wantLength: 4},
		{name: "rocket", codes: []string{"BJ", "RJ"}, wantType: CardGroupTypeRocket, wantPrimary: RankRedJoker, wantLength: 2},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cards := mustCards(t, tc.codes)
			group, err := Recognize(cards)
			if err != nil {
				t.Fatalf("Recognize() error = %v", err)
			}
			if group.Type != tc.wantType {
				t.Fatalf("Type = %s, want %s", group.Type, tc.wantType)
			}
			if group.PrimaryRank != tc.wantPrimary {
				t.Fatalf("PrimaryRank = %s, want %s", group.PrimaryRank, tc.wantPrimary)
			}
			if group.Length != tc.wantLength {
				t.Fatalf("Length = %d, want %d", group.Length, tc.wantLength)
			}
			if len(group.Attachments) != tc.wantAttach {
				t.Fatalf("Attachments len = %d, want %d", len(group.Attachments), tc.wantAttach)
			}
			if len(group.Cards) != len(cards) {
				t.Fatalf("Cards len = %d, want %d", len(group.Cards), len(cards))
			}
		})
	}
}

func TestRecognizeInvalidGroups(t *testing.T) {
	testCases := [][]string{
		{},
		{"S3", "S3"},
		{"S3", "H4"},
		{"S3", "H3", "D4"},
		{"S3", "H3", "D3", "S3"},
		{"S3", "H4", "D5", "S6"},
		{"S3", "H3", "S4", "H4"},
		{"S3", "H3", "D3", "S4", "H5"},
		{"S3", "H3", "D3", "S4", "H4", "D4", "S5"},
		{"S3", "H3", "D3", "C3", "S4", "H4", "S5", "H6"},
		{"S2", "H3", "D4", "S5", "H6"},
		{"BJ", "S3"},
	}

	for _, codes := range testCases {
		codes := codes
		t.Run(joinCodes(codes), func(t *testing.T) {
			cards := mustCards(t, codes)
			_, err := Recognize(cards)
			if !errors.Is(err, ErrInvalidCardSet) {
				t.Fatalf("Recognize() error = %v, want ErrInvalidCardSet", err)
			}
		})
	}
}

func TestRecognizeRejectsDuplicateCards(t *testing.T) {
	cards := mustCards(t, []string{"S3", "S3"})
	_, err := Recognize(cards)
	if !errors.Is(err, ErrInvalidCardSet) {
		t.Fatalf("Recognize() error = %v, want ErrInvalidCardSet", err)
	}
}

func TestRecognizeReturnsCopiesOfCards(t *testing.T) {
	cards := mustCards(t, []string{"S3", "H3"})
	group, err := Recognize(cards)
	if err != nil {
		t.Fatalf("Recognize() error = %v", err)
	}
	if !reflect.DeepEqual(group.Cards, cards) {
		t.Fatalf("group cards = %#v, want %#v", group.Cards, cards)
	}
}

func mustCards(t *testing.T, codes []string) []Card {
	t.Helper()
	cards := make([]Card, 0, len(codes))
	for _, code := range codes {
		card, err := ParseCard(code)
		if err != nil {
			t.Fatalf("ParseCard(%q) error = %v", code, err)
		}
		cards = append(cards, card)
	}
	return cards
}

func joinCodes(codes []string) string {
	if len(codes) == 0 {
		return "empty"
	}
	return codes[0]
}
