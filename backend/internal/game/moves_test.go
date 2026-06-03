package game

import (
	"errors"
	"testing"
)

func TestContainsCards(t *testing.T) {
	hand := mustCards(t, []string{"S3", "H3", "D4", "BJ"})

	if !ContainsCards(hand, mustCards(t, []string{"S3", "BJ"})) {
		t.Fatal("expected cards to be contained in hand")
	}
	if ContainsCards(hand, mustCards(t, []string{"RJ"})) {
		t.Fatal("expected missing card to return false")
	}
	if ContainsCards(hand, mustCards(t, []string{"S3", "S3"})) {
		t.Fatal("expected duplicate request to fail when hand lacks duplicates")
	}
}

func TestRemoveCards(t *testing.T) {
	hand := mustCards(t, []string{"S3", "H3", "D4", "BJ"})

	remaining, err := RemoveCards(hand, mustCards(t, []string{"H3", "BJ"}))
	if err != nil {
		t.Fatalf("RemoveCards() error = %v", err)
	}
	assertGroupCodes(t, remaining, []string{"S3", "D4"})

	_, err = RemoveCards(hand, mustCards(t, []string{"RJ"}))
	if !errors.Is(err, ErrInvalidCardSet) {
		t.Fatalf("RemoveCards() error = %v, want ErrInvalidCardSet", err)
	}
}

func TestLegalMovesOpeningContainsExpectedGroups(t *testing.T) {
	hand := mustCards(t, []string{
		"S3", "H3", "D3",
		"S4", "H4",
		"S5", "H5",
		"S6", "H6",
		"S7",
		"BJ", "RJ",
	})

	moves := LegalMoves(hand, nil)

	assertHasGroup(t, moves, CardGroupTypeSingle, Rank3, 1)
	assertHasGroup(t, moves, CardGroupTypePair, Rank3, 2)
	assertHasGroup(t, moves, CardGroupTypeTriple, Rank3, 3)
	assertHasGroup(t, moves, CardGroupTypeTripleWithSingle, Rank3, 4)
	assertHasGroup(t, moves, CardGroupTypeTripleWithPair, Rank3, 5)
	assertHasGroup(t, moves, CardGroupTypeStraight, Rank7, 5)
	assertHasGroup(t, moves, CardGroupTypePairStraight, Rank6, 3)
	assertHasGroup(t, moves, CardGroupTypeRocket, RankRedJoker, 2)
}

func TestLegalResponsesFiltersByPreviousGroup(t *testing.T) {
	hand := mustCards(t, []string{
		"S5", "H5",
		"S6", "H6",
		"S7", "H7", "D7", "C7",
		"BJ", "RJ",
	})

	previous := mustRecognize(t, []string{"S4", "H4"})
	moves := LegalResponses(hand, &previous)

	assertHasGroup(t, moves, CardGroupTypePair, Rank5, 2)
	assertHasGroup(t, moves, CardGroupTypePair, Rank6, 2)
	assertHasGroup(t, moves, CardGroupTypeBomb, Rank7, 4)
	assertHasGroup(t, moves, CardGroupTypeRocket, RankRedJoker, 2)
	assertNoGroup(t, moves, CardGroupTypeSingle, Rank5, 1)
}

func TestLegalResponsesAgainstBombOnlyReturnsHigherBombOrRocket(t *testing.T) {
	hand := mustCards(t, []string{
		"S8", "H8", "D8", "C8",
		"BJ", "RJ",
		"S9", "H9",
	})

	previous := mustRecognize(t, []string{"S7", "H7", "D7", "C7"})
	moves := LegalResponses(hand, &previous)

	assertHasGroup(t, moves, CardGroupTypeBomb, Rank8, 4)
	assertHasGroup(t, moves, CardGroupTypeRocket, RankRedJoker, 2)
	assertNoGroup(t, moves, CardGroupTypePair, Rank9, 2)
}

func assertHasGroup(t *testing.T, groups []CardGroup, wantType CardGroupType, wantPrimary Rank, wantLength int) {
	t.Helper()
	for _, group := range groups {
		if group.Type == wantType && group.PrimaryRank == wantPrimary && group.Length == wantLength {
			return
		}
	}
	t.Fatalf("group not found: type=%s primary=%s length=%d", wantType, wantPrimary, wantLength)
}

func assertNoGroup(t *testing.T, groups []CardGroup, wantType CardGroupType, wantPrimary Rank, wantLength int) {
	t.Helper()
	for _, group := range groups {
		if group.Type == wantType && group.PrimaryRank == wantPrimary && group.Length == wantLength {
			t.Fatalf("unexpected group found: type=%s primary=%s length=%d", wantType, wantPrimary, wantLength)
		}
	}
}

func assertGroupCodes(t *testing.T, cards []Card, want []string) {
	t.Helper()
	if len(cards) != len(want) {
		t.Fatalf("len(cards) = %d, want %d", len(cards), len(want))
	}
	for i, card := range cards {
		if card.Code() != want[i] {
			t.Fatalf("cards[%d] = %s, want %s", i, card.Code(), want[i])
		}
	}
}
