package game

import (
	"errors"
	"fmt"
	"sort"
)

var ErrInvalidCardSet = errors.New("invalid card set")

type CardGroupType string

const (
	CardGroupTypeSingle              CardGroupType = "single"
	CardGroupTypePair                CardGroupType = "pair"
	CardGroupTypeTriple              CardGroupType = "triple"
	CardGroupTypeTripleWithSingle    CardGroupType = "triple_with_single"
	CardGroupTypeTripleWithPair      CardGroupType = "triple_with_pair"
	CardGroupTypeStraight            CardGroupType = "straight"
	CardGroupTypePairStraight        CardGroupType = "pair_straight"
	CardGroupTypeAirplane            CardGroupType = "airplane"
	CardGroupTypeAirplaneWithSingles CardGroupType = "airplane_with_singles"
	CardGroupTypeAirplaneWithPairs   CardGroupType = "airplane_with_pairs"
	CardGroupTypeFourWithTwoSingles  CardGroupType = "four_with_two_singles"
	CardGroupTypeFourWithTwoPairs    CardGroupType = "four_with_two_pairs"
	CardGroupTypeBomb                CardGroupType = "bomb"
	CardGroupTypeRocket              CardGroupType = "rocket"
)

type CardGroup struct {
	Type        CardGroupType
	PrimaryRank Rank
	Cards       []Card
	Length      int
	Attachments []Card
}

func Recognize(cards []Card) (CardGroup, error) {
	if len(cards) == 0 {
		return CardGroup{}, fmt.Errorf("%w: empty set", ErrInvalidCardSet)
	}

	if err := validateUniqueCards(cards); err != nil {
		return CardGroup{}, err
	}

	sorted := append([]Card(nil), cards...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rank != sorted[j].Rank {
			return sorted[i].Rank < sorted[j].Rank
		}
		return sorted[i].Suit < sorted[j].Suit
	})

	if len(sorted) == 2 && isRocket(sorted) {
		return CardGroup{
			Type:        CardGroupTypeRocket,
			PrimaryRank: RankRedJoker,
			Cards:       copyCards(cards),
			Length:      2,
		}, nil
	}

	counts := countByRank(sorted)
	switch len(sorted) {
	case 1:
		return CardGroup{
			Type:        CardGroupTypeSingle,
			PrimaryRank: sorted[0].Rank,
			Cards:       copyCards(cards),
			Length:      1,
		}, nil
	case 2:
		if hasCount(counts, 2) {
			return CardGroup{
				Type:        CardGroupTypePair,
				PrimaryRank: sorted[0].Rank,
				Cards:       copyCards(cards),
				Length:      2,
			}, nil
		}
	case 3:
		if hasCount(counts, 3) {
			return CardGroup{
				Type:        CardGroupTypeTriple,
				PrimaryRank: sorted[0].Rank,
				Cards:       copyCards(cards),
				Length:      3,
			}, nil
		}
	case 4:
		if hasCount(counts, 4) {
			return CardGroup{
				Type:        CardGroupTypeBomb,
				PrimaryRank: sorted[0].Rank,
				Cards:       copyCards(cards),
				Length:      4,
			}, nil
		}
		if tripleRank, ok := findCountRank(counts, 3); ok && hasCount(counts, 1) {
			attachments := extractCardsByRank(sorted, func(card Card) bool {
				return card.Rank != tripleRank
			})
			return CardGroup{
				Type:        CardGroupTypeTripleWithSingle,
				PrimaryRank: tripleRank,
				Cards:       copyCards(cards),
				Length:      4,
				Attachments: attachments,
			}, nil
		}
	case 5:
		if tripleRank, ok := findCountRank(counts, 3); ok && hasCount(counts, 2) {
			attachments := extractCardsByRank(sorted, func(card Card) bool {
				return card.Rank != tripleRank
			})
			return CardGroup{
				Type:        CardGroupTypeTripleWithPair,
				PrimaryRank: tripleRank,
				Cards:       copyCards(cards),
				Length:      5,
				Attachments: attachments,
			}, nil
		}
	}

	if group, ok := recognizeStraight(cards, counts); ok {
		return group, nil
	}
	if group, ok := recognizePairStraight(cards, counts); ok {
		return group, nil
	}
	if group, ok := recognizeAirplane(cards, counts); ok {
		return group, nil
	}
	if group, ok := recognizeAirplaneWithAttachments(cards, counts, false); ok {
		return group, nil
	}
	if group, ok := recognizeAirplaneWithAttachments(cards, counts, true); ok {
		return group, nil
	}
	if group, ok := recognizeFourWithAttachments(cards, counts, false); ok {
		return group, nil
	}
	if group, ok := recognizeFourWithAttachments(cards, counts, true); ok {
		return group, nil
	}

	return CardGroup{}, fmt.Errorf("%w: %s", ErrInvalidCardSet, cardsToCodes(sorted))
}

func validateUniqueCards(cards []Card) error {
	seen := make(map[string]struct{}, len(cards))
	for _, card := range cards {
		code := card.Code()
		if _, exists := seen[code]; exists {
			return fmt.Errorf("%w: duplicate card %s", ErrInvalidCardSet, code)
		}
		seen[code] = struct{}{}
	}
	return nil
}

func isRocket(cards []Card) bool {
	if len(cards) != 2 {
		return false
	}
	ranks := []Rank{cards[0].Rank, cards[1].Rank}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })
	return ranks[0] == RankBlackJoker && ranks[1] == RankRedJoker
}

func countByRank(cards []Card) map[Rank]int {
	counts := make(map[Rank]int, len(cards))
	for _, card := range cards {
		counts[card.Rank]++
	}
	return counts
}

func hasCount(counts map[Rank]int, target int) bool {
	for _, count := range counts {
		if count == target {
			return true
		}
	}
	return false
}

func findCountRank(counts map[Rank]int, target int) (Rank, bool) {
	var found Rank
	ok := false
	for rank, count := range counts {
		if count == target {
			found = rank
			ok = true
			break
		}
	}
	return found, ok
}

func recognizeStraight(cards []Card, counts map[Rank]int) (CardGroup, bool) {
	if len(cards) < 5 {
		return CardGroup{}, false
	}
	if !allCountsEqual(counts, 1) {
		return CardGroup{}, false
	}

	ranks := uniqueSortedRanks(cards)
	if len(ranks) != len(cards) {
		return CardGroup{}, false
	}
	if !isConsecutive(ranks) || ranks[len(ranks)-1] > RankA {
		return CardGroup{}, false
	}

	return CardGroup{
		Type:        CardGroupTypeStraight,
		PrimaryRank: ranks[len(ranks)-1],
		Cards:       copyCards(cards),
		Length:      len(cards),
	}, true
}

func recognizePairStraight(cards []Card, counts map[Rank]int) (CardGroup, bool) {
	if len(cards) < 6 || len(cards)%2 != 0 {
		return CardGroup{}, false
	}
	if !allCountsEqual(counts, 2) {
		return CardGroup{}, false
	}

	ranks := uniqueSortedRanks(cards)
	if len(ranks)*2 != len(cards) {
		return CardGroup{}, false
	}
	if !isConsecutive(ranks) || ranks[len(ranks)-1] > RankA {
		return CardGroup{}, false
	}

	return CardGroup{
		Type:        CardGroupTypePairStraight,
		PrimaryRank: ranks[len(ranks)-1],
		Cards:       copyCards(cards),
		Length:      len(cards) / 2,
	}, true
}

func recognizeAirplane(cards []Card, counts map[Rank]int) (CardGroup, bool) {
	if len(cards) < 6 || len(cards)%3 != 0 {
		return CardGroup{}, false
	}
	if !allCountsEqual(counts, 3) {
		return CardGroup{}, false
	}

	ranks := uniqueSortedRanks(cards)
	if len(ranks)*3 != len(cards) {
		return CardGroup{}, false
	}
	if !isConsecutive(ranks) || ranks[len(ranks)-1] > RankA {
		return CardGroup{}, false
	}

	return CardGroup{
		Type:        CardGroupTypeAirplane,
		PrimaryRank: ranks[len(ranks)-1],
		Cards:       copyCards(cards),
		Length:      len(cards) / 3,
	}, true
}

func recognizeAirplaneWithAttachments(cards []Card, counts map[Rank]int, withPairs bool) (CardGroup, bool) {
	baseCount := 4
	if withPairs {
		baseCount = 5
	}
	if len(cards) < baseCount*2 || len(cards)%baseCount != 0 {
		return CardGroup{}, false
	}

	tripleRanks := ranksWithCount(counts, 3)
	if len(tripleRanks) < 2 {
		return CardGroup{}, false
	}
	if !isConsecutive(tripleRanks) || tripleRanks[len(tripleRanks)-1] > RankA {
		return CardGroup{}, false
	}

	triplesNeeded := len(tripleRanks)
	if withPairs {
		if len(cards) != triplesNeeded*5 {
			return CardGroup{}, false
		}
	} else {
		if len(cards) != triplesNeeded*4 {
			return CardGroup{}, false
		}
	}

	attachmentCounts := make(map[Rank]int)
	for rank, count := range counts {
		switch count {
		case 3:
			continue
		case 1:
			if withPairs {
				return CardGroup{}, false
			}
		case 2:
			if !withPairs {
				return CardGroup{}, false
			}
		default:
			return CardGroup{}, false
		}
		attachmentCounts[rank] = count
	}

	if len(attachmentCounts) != triplesNeeded {
		return CardGroup{}, false
	}

	groupType := CardGroupTypeAirplaneWithSingles
	if withPairs {
		groupType = CardGroupTypeAirplaneWithPairs
	}
	attachments := extractCardsByRank(cards, func(card Card) bool {
		return counts[card.Rank] != 3
	})

	return CardGroup{
		Type:        groupType,
		PrimaryRank: tripleRanks[len(tripleRanks)-1],
		Cards:       copyCards(cards),
		Length:      triplesNeeded,
		Attachments: attachments,
	}, true
}

func recognizeFourWithAttachments(cards []Card, counts map[Rank]int, withPairs bool) (CardGroup, bool) {
	if len(cards) != 6 && len(cards) != 8 {
		return CardGroup{}, false
	}

	fourRank, ok := findCountRank(counts, 4)
	if !ok {
		return CardGroup{}, false
	}

	switch len(cards) {
	case 6:
		if withPairs {
			return CardGroup{}, false
		}
		if !allOtherCounts(counts, fourRank, 1) {
			return CardGroup{}, false
		}
		attachments := extractCardsByRank(cards, func(card Card) bool {
			return card.Rank != fourRank
		})
		return CardGroup{
			Type:        CardGroupTypeFourWithTwoSingles,
			PrimaryRank: fourRank,
			Cards:       copyCards(cards),
			Length:      6,
			Attachments: attachments,
		}, true
	case 8:
		if !withPairs {
			return CardGroup{}, false
		}
		if !allOtherCounts(counts, fourRank, 2) {
			return CardGroup{}, false
		}
		attachments := extractCardsByRank(cards, func(card Card) bool {
			return card.Rank != fourRank
		})
		return CardGroup{
			Type:        CardGroupTypeFourWithTwoPairs,
			PrimaryRank: fourRank,
			Cards:       copyCards(cards),
			Length:      8,
			Attachments: attachments,
		}, true
	default:
		return CardGroup{}, false
	}
}

func allCountsEqual(counts map[Rank]int, expected int) bool {
	for _, count := range counts {
		if count != expected {
			return false
		}
	}
	return len(counts) > 0
}

func allOtherCounts(counts map[Rank]int, excluded Rank, expected int) bool {
	for rank, count := range counts {
		if rank == excluded {
			continue
		}
		if count != expected {
			return false
		}
	}
	return true
}

func uniqueSortedRanks(cards []Card) []Rank {
	ranks := make([]Rank, 0, len(cards))
	seen := make(map[Rank]struct{}, len(cards))
	for _, card := range cards {
		if _, ok := seen[card.Rank]; ok {
			continue
		}
		seen[card.Rank] = struct{}{}
		ranks = append(ranks, card.Rank)
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })
	return ranks
}

func ranksWithCount(counts map[Rank]int, expected int) []Rank {
	ranks := make([]Rank, 0, len(counts))
	for rank, count := range counts {
		if count == expected {
			ranks = append(ranks, rank)
		}
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })
	return ranks
}

func isConsecutive(ranks []Rank) bool {
	if len(ranks) == 0 {
		return false
	}
	for i := 1; i < len(ranks); i++ {
		if ranks[i] != ranks[i-1]+1 {
			return false
		}
	}
	return true
}

func extractCardsByRank(cards []Card, keep func(Card) bool) []Card {
	result := make([]Card, 0, len(cards))
	for _, card := range cards {
		if keep(card) {
			result = append(result, card)
		}
	}
	return result
}

func copyCards(cards []Card) []Card {
	return append([]Card(nil), cards...)
}

func cardsToCodes(cards []Card) []string {
	codes := make([]string, 0, len(cards))
	for _, card := range cards {
		codes = append(codes, card.Code())
	}
	return codes
}
