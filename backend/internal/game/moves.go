package game

import (
	"fmt"
	"sort"
)

func ContainsCards(hand []Card, cards []Card) bool {
	if len(cards) == 0 {
		return true
	}

	available := make(map[string]int, len(hand))
	for _, card := range hand {
		available[card.Code()]++
	}

	for _, card := range cards {
		code := card.Code()
		if available[code] == 0 {
			return false
		}
		available[code]--
	}

	return true
}

func RemoveCards(hand []Card, cards []Card) ([]Card, error) {
	if !ContainsCards(hand, cards) {
		return nil, fmt.Errorf("%w: cards not in hand", ErrInvalidCardSet)
	}

	toRemove := make(map[string]int, len(cards))
	for _, card := range cards {
		toRemove[card.Code()]++
	}

	remaining := make([]Card, 0, len(hand)-len(cards))
	for _, card := range hand {
		code := card.Code()
		if toRemove[code] > 0 {
			toRemove[code]--
			continue
		}
		remaining = append(remaining, card)
	}

	return remaining, nil
}

func LegalResponses(hand []Card, previous *CardGroup) []CardGroup {
	return LegalMoves(hand, previous)
}

func LegalMoves(hand []Card, previous *CardGroup) []CardGroup {
	groups := generateAllGroups(hand)
	if previous == nil || previous.Type == "" {
		return groups
	}

	filtered := make([]CardGroup, 0, len(groups))
	for _, group := range groups {
		if CanBeat(group, *previous) {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func generateAllGroups(hand []Card) []CardGroup {
	index := newHandIndex(hand)
	collector := newGroupCollector()

	addSingles(index, collector)
	addPairs(index, collector)
	addTriples(index, collector)
	addTripleWithSingles(index, collector)
	addTripleWithPairs(index, collector)
	addStraights(index, collector)
	addPairStraights(index, collector)
	addAirplanes(index, collector)
	addAirplanesWithSingles(index, collector)
	addAirplanesWithPairs(index, collector)
	addFourWithTwoSingles(index, collector)
	addFourWithTwoPairs(index, collector)
	addBombs(index, collector)
	addRocket(index, collector)

	return collector.Groups()
}

type handIndex struct {
	byRank map[Rank][]Card
	ranks  []Rank
}

func newHandIndex(hand []Card) handIndex {
	byRank := make(map[Rank][]Card)
	for _, card := range hand {
		byRank[card.Rank] = append(byRank[card.Rank], card)
	}

	ranks := make([]Rank, 0, len(byRank))
	for rank := range byRank {
		sort.Slice(byRank[rank], func(i, j int) bool {
			return byRank[rank][i].Suit < byRank[rank][j].Suit
		})
		ranks = append(ranks, rank)
	}

	sort.Slice(ranks, func(i, j int) bool { return ranks[i] < ranks[j] })

	return handIndex{
		byRank: byRank,
		ranks:  ranks,
	}
}

func (h handIndex) ranksAtLeast(n int, max Rank) []Rank {
	result := make([]Rank, 0, len(h.ranks))
	for _, rank := range h.ranks {
		if rank > max {
			continue
		}
		if len(h.byRank[rank]) >= n {
			result = append(result, rank)
		}
	}
	return result
}

func (h handIndex) ranksExcluding(excluded map[Rank]struct{}, minCount int) []Rank {
	result := make([]Rank, 0, len(h.ranks))
	for _, rank := range h.ranks {
		if _, exists := excluded[rank]; exists {
			continue
		}
		if len(h.byRank[rank]) >= minCount {
			result = append(result, rank)
		}
	}
	return result
}

func (h handIndex) take(rank Rank, count int) []Card {
	cards := h.byRank[rank]
	return append([]Card(nil), cards[:count]...)
}

type groupCollector struct {
	seen   map[string]struct{}
	groups []CardGroup
}

func newGroupCollector() *groupCollector {
	return &groupCollector{
		seen: make(map[string]struct{}),
	}
}

func (c *groupCollector) Add(cards []Card) {
	group, err := Recognize(cards)
	if err != nil {
		return
	}

	key := groupKey(group)
	if _, exists := c.seen[key]; exists {
		return
	}

	c.seen[key] = struct{}{}
	c.groups = append(c.groups, group)
}

func (c *groupCollector) Groups() []CardGroup {
	sort.Slice(c.groups, func(i, j int) bool {
		if groupOrder(c.groups[i].Type) != groupOrder(c.groups[j].Type) {
			return groupOrder(c.groups[i].Type) < groupOrder(c.groups[j].Type)
		}
		if c.groups[i].Length != c.groups[j].Length {
			return c.groups[i].Length < c.groups[j].Length
		}
		if c.groups[i].PrimaryRank != c.groups[j].PrimaryRank {
			return c.groups[i].PrimaryRank < c.groups[j].PrimaryRank
		}
		return groupKey(c.groups[i]) < groupKey(c.groups[j])
	})
	return append([]CardGroup(nil), c.groups...)
}

func addSingles(index handIndex, collector *groupCollector) {
	for _, rank := range index.ranks {
		collector.Add(index.take(rank, 1))
	}
}

func addPairs(index handIndex, collector *groupCollector) {
	for _, rank := range index.ranksAtLeast(2, RankRedJoker) {
		collector.Add(index.take(rank, 2))
	}
}

func addTriples(index handIndex, collector *groupCollector) {
	for _, rank := range index.ranksAtLeast(3, Rank2) {
		collector.Add(index.take(rank, 3))
	}
}

func addTripleWithSingles(index handIndex, collector *groupCollector) {
	triples := index.ranksAtLeast(3, Rank2)
	for _, tripleRank := range triples {
		excluded := map[Rank]struct{}{tripleRank: {}}
		for _, singleRank := range index.ranksExcluding(excluded, 1) {
			cards := append(index.take(tripleRank, 3), index.take(singleRank, 1)...)
			collector.Add(cards)
		}
	}
}

func addTripleWithPairs(index handIndex, collector *groupCollector) {
	triples := index.ranksAtLeast(3, Rank2)
	for _, tripleRank := range triples {
		excluded := map[Rank]struct{}{tripleRank: {}}
		for _, pairRank := range index.ranksExcluding(excluded, 2) {
			cards := append(index.take(tripleRank, 3), index.take(pairRank, 2)...)
			collector.Add(cards)
		}
	}
}

func addStraights(index handIndex, collector *groupCollector) {
	ranks := index.ranksAtLeast(1, RankA)
	for _, sequence := range consecutiveWindows(ranks, 5) {
		for length := 5; length <= len(sequence); length++ {
			for start := 0; start+length <= len(sequence); start++ {
				window := sequence[start : start+length]
				var cards []Card
				for _, rank := range window {
					cards = append(cards, index.take(rank, 1)...)
				}
				collector.Add(cards)
			}
		}
	}
}

func addPairStraights(index handIndex, collector *groupCollector) {
	ranks := index.ranksAtLeast(2, RankA)
	for _, sequence := range consecutiveWindows(ranks, 3) {
		for length := 3; length <= len(sequence); length++ {
			for start := 0; start+length <= len(sequence); start++ {
				window := sequence[start : start+length]
				var cards []Card
				for _, rank := range window {
					cards = append(cards, index.take(rank, 2)...)
				}
				collector.Add(cards)
			}
		}
	}
}

func addAirplanes(index handIndex, collector *groupCollector) {
	ranks := index.ranksAtLeast(3, RankA)
	for _, sequence := range consecutiveWindows(ranks, 2) {
		for length := 2; length <= len(sequence); length++ {
			for start := 0; start+length <= len(sequence); start++ {
				window := sequence[start : start+length]
				var cards []Card
				for _, rank := range window {
					cards = append(cards, index.take(rank, 3)...)
				}
				collector.Add(cards)
			}
		}
	}
}

func addAirplanesWithSingles(index handIndex, collector *groupCollector) {
	ranks := index.ranksAtLeast(3, RankA)
	for _, sequence := range consecutiveWindows(ranks, 2) {
		for length := 2; length <= len(sequence); length++ {
			for start := 0; start+length <= len(sequence); start++ {
				window := sequence[start : start+length]
				excluded := sliceToRankSet(window)
				pool := index.cardsExcludingRanks(excluded)
				for _, attachCards := range chooseCardCombos(pool, length) {
					var cards []Card
					for _, rank := range window {
						cards = append(cards, index.take(rank, 3)...)
					}
					cards = append(cards, attachCards...)
					collector.Add(cards)
				}
			}
		}
	}
}

func addAirplanesWithPairs(index handIndex, collector *groupCollector) {
	ranks := index.ranksAtLeast(3, RankA)
	for _, sequence := range consecutiveWindows(ranks, 2) {
		for length := 2; length <= len(sequence); length++ {
			for start := 0; start+length <= len(sequence); start++ {
				window := sequence[start : start+length]
				excluded := sliceToRankSet(window)
				attachments := chooseRanks(index.ranksExcluding(excluded, 2), length)
				for _, attachRanks := range attachments {
					var cards []Card
					for _, rank := range window {
						cards = append(cards, index.take(rank, 3)...)
					}
					for _, rank := range attachRanks {
						cards = append(cards, index.take(rank, 2)...)
					}
					collector.Add(cards)
				}
			}
		}
	}
}

func addFourWithTwoSingles(index handIndex, collector *groupCollector) {
	fours := index.ranksAtLeast(4, Rank2)
	for _, fourRank := range fours {
		excluded := map[Rank]struct{}{fourRank: {}}
		pool := index.cardsExcludingRanks(excluded)
		for _, attachCards := range chooseCardCombos(pool, 2) {
			cards := append([]Card(nil), index.take(fourRank, 4)...)
			cards = append(cards, attachCards...)
			collector.Add(cards)
		}
	}
}

func addFourWithTwoPairs(index handIndex, collector *groupCollector) {
	fours := index.ranksAtLeast(4, Rank2)
	for _, fourRank := range fours {
		excluded := map[Rank]struct{}{fourRank: {}}
		for _, attachRanks := range chooseRanks(index.ranksExcluding(excluded, 2), 2) {
			cards := append([]Card(nil), index.take(fourRank, 4)...)
			for _, rank := range attachRanks {
				cards = append(cards, index.take(rank, 2)...)
			}
			collector.Add(cards)
		}
	}
}

func addBombs(index handIndex, collector *groupCollector) {
	for _, rank := range index.ranksAtLeast(4, Rank2) {
		collector.Add(index.take(rank, 4))
	}
}

func addRocket(index handIndex, collector *groupCollector) {
	if len(index.byRank[RankBlackJoker]) == 1 && len(index.byRank[RankRedJoker]) == 1 {
		cards := append(index.take(RankBlackJoker, 1), index.take(RankRedJoker, 1)...)
		collector.Add(cards)
	}
}

func consecutiveWindows(ranks []Rank, minLength int) [][]Rank {
	if len(ranks) < minLength {
		return nil
	}

	var result [][]Rank
	start := 0
	for start < len(ranks) {
		end := start + 1
		for end < len(ranks) && ranks[end] == ranks[end-1]+1 {
			end++
		}
		if end-start >= minLength {
			result = append(result, append([]Rank(nil), ranks[start:end]...))
		}
		start = end
	}
	return result
}

func chooseRanks(ranks []Rank, want int) [][]Rank {
	if want == 0 {
		return [][]Rank{{}}
	}
	if len(ranks) < want {
		return nil
	}

	var result [][]Rank
	var current []Rank

	var dfs func(start int)
	dfs = func(start int) {
		if len(current) == want {
			result = append(result, append([]Rank(nil), current...))
			return
		}
		for i := start; i <= len(ranks)-(want-len(current)); i++ {
			current = append(current, ranks[i])
			dfs(i + 1)
			current = current[:len(current)-1]
		}
	}

	dfs(0)
	return result
}

func sliceToRankSet(ranks []Rank) map[Rank]struct{} {
	set := make(map[Rank]struct{}, len(ranks))
	for _, rank := range ranks {
		set[rank] = struct{}{}
	}
	return set
}

func (h handIndex) cardsExcludingRanks(excluded map[Rank]struct{}) []Card {
	var cards []Card
	for _, rank := range h.ranks {
		if _, exists := excluded[rank]; exists {
			continue
		}
		cards = append(cards, h.byRank[rank]...)
	}
	return cards
}

func chooseCardCombos(cards []Card, want int) [][]Card {
	if want == 0 {
		return [][]Card{{}}
	}
	if len(cards) < want {
		return nil
	}

	var result [][]Card
	var current []Card

	var dfs func(start int)
	dfs = func(start int) {
		if len(current) == want {
			result = append(result, append([]Card(nil), current...))
			return
		}
		for i := start; i <= len(cards)-(want-len(current)); i++ {
			current = append(current, cards[i])
			dfs(i + 1)
			current = current[:len(current)-1]
		}
	}

	dfs(0)
	return result
}

func groupKey(group CardGroup) string {
	codes := cardsToCodes(group.Cards)
	sort.Strings(codes)
	return string(group.Type) + "|" + fmt.Sprint(group.PrimaryRank) + "|" + fmt.Sprint(group.Length) + "|" + fmt.Sprint(codes)
}

func groupOrder(groupType CardGroupType) int {
	switch groupType {
	case CardGroupTypeSingle:
		return 1
	case CardGroupTypePair:
		return 2
	case CardGroupTypeTriple:
		return 3
	case CardGroupTypeTripleWithSingle:
		return 4
	case CardGroupTypeTripleWithPair:
		return 5
	case CardGroupTypeStraight:
		return 6
	case CardGroupTypePairStraight:
		return 7
	case CardGroupTypeAirplane:
		return 8
	case CardGroupTypeAirplaneWithSingles:
		return 9
	case CardGroupTypeAirplaneWithPairs:
		return 10
	case CardGroupTypeFourWithTwoSingles:
		return 11
	case CardGroupTypeFourWithTwoPairs:
		return 12
	case CardGroupTypeBomb:
		return 13
	case CardGroupTypeRocket:
		return 14
	default:
		return 99
	}
}
