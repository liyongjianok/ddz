package room

import (
	"sort"
	"strings"

	"ddz/backend/internal/game"
)

type robotPlayChoice struct {
	cards []game.Card
	pass  bool
}

// chooseRobotBid 根据手牌强度和当前最高叫分选择合法叫分。
func chooseRobotBid(currentGame *game.Game, seatIndex int) int {
	if currentGame == nil || seatIndex < 0 || seatIndex >= len(currentGame.Players) {
		return 0
	}

	strength := robotHandStrength(currentGame.Players[seatIndex].Hand)
	wantScore := 0
	switch {
	case strength >= 16:
		wantScore = 3
	case strength >= 12:
		wantScore = 2
	case strength >= 8:
		wantScore = 1
	default:
		return 0
	}

	if wantScore <= currentGame.BiddingState.HighestBid {
		return 0
	}
	return wantScore
}

// chooseRobotPlay 基于公开状态和自己手牌选择出牌或不出。
func chooseRobotPlay(currentGame *game.Game, seatIndex int) robotPlayChoice {
	if currentGame == nil || seatIndex < 0 || seatIndex >= len(currentGame.Players) {
		return robotPlayChoice{pass: true}
	}

	player := currentGame.Players[seatIndex]
	hand := player.Hand
	if len(hand) == 0 {
		return robotPlayChoice{pass: true}
	}

	if currentGame.LastPlay == nil {
		return chooseRobotLead(currentGame, seatIndex, hand)
	}

	previous := currentGame.LastPlay.Group
	moves := game.LegalResponses(hand, &previous)
	if len(moves) == 0 {
		return robotPlayChoice{pass: true}
	}
	if winningMove, ok := findWinningMove(moves, len(hand)); ok {
		return robotPlayChoice{cards: winningMove.Cards}
	}
	if isFarmerTeammatePlay(currentGame, seatIndex, currentGame.LastPlay.SeatIndex) {
		return robotPlayChoice{pass: true}
	}

	nonSpecial := filterGroups(moves, func(group game.CardGroup) bool {
		return !isBombOrRocket(group)
	})
	if len(nonSpecial) > 0 {
		return robotPlayChoice{cards: chooseSmallestResponse(nonSpecial).Cards}
	}
	if hasOpponentWithFewCards(currentGame, seatIndex, 2) {
		return robotPlayChoice{cards: chooseSmallestResponse(moves).Cards}
	}

	return robotPlayChoice{pass: true}
}

func chooseRobotLead(currentGame *game.Game, seatIndex int, hand []game.Card) robotPlayChoice {
	moves := game.LegalMoves(hand, nil)
	if len(moves) == 0 {
		return robotPlayChoice{pass: true}
	}
	if winningMove, ok := findWinningMove(moves, len(hand)); ok {
		return robotPlayChoice{cards: winningMove.Cards}
	}

	candidates := filterGroups(moves, func(group game.CardGroup) bool {
		return !isBombOrRocket(group)
	})
	if len(candidates) == 0 {
		candidates = moves
	}

	nextSeat := (seatIndex + 1) % game.PlayerCount
	if currentGame.Players[nextSeat].RemainingCount == 1 {
		withoutSingles := filterGroups(candidates, func(group game.CardGroup) bool {
			return group.Type != game.CardGroupTypeSingle
		})
		if len(withoutSingles) > 0 {
			candidates = withoutSingles
		}
	}
	if currentGame.Players[nextSeat].RemainingCount == 2 {
		withoutPairs := filterGroups(candidates, func(group game.CardGroup) bool {
			return group.Type != game.CardGroupTypePair
		})
		if len(withoutPairs) > 0 {
			candidates = withoutPairs
		}
	}

	if robotHandStrength(hand) < 8 {
		if single, ok := findLowestGroup(candidates, game.CardGroupTypeSingle); ok {
			return robotPlayChoice{cards: single.Cards}
		}
	}

	return robotPlayChoice{cards: chooseBestLead(candidates).Cards}
}

func robotHandStrength(hand []game.Card) int {
	counts := make(map[game.Rank]int, len(hand))
	for _, card := range hand {
		counts[card.Rank]++
	}

	score := 0
	if counts[game.RankBlackJoker] > 0 && counts[game.RankRedJoker] > 0 {
		score += 8
	}
	for rank, count := range counts {
		if count == 4 {
			score += 6
		}
		switch rank {
		case game.Rank2:
			score += 2 * count
		case game.RankBlackJoker, game.RankRedJoker:
			score += 3 * count
		case game.RankA, game.RankK:
			score += count
		}
	}

	hasLongStraight := false
	hasAirplane := false
	for _, group := range game.LegalMoves(hand, nil) {
		switch group.Type {
		case game.CardGroupTypeStraight:
			if group.Length >= 5 {
				hasLongStraight = true
			}
		case game.CardGroupTypeAirplane, game.CardGroupTypeAirplaneWithSingles, game.CardGroupTypeAirplaneWithPairs:
			if group.Length >= 2 {
				hasAirplane = true
			}
		}
	}
	if hasLongStraight {
		score += 2
	}
	if hasAirplane {
		score += 2
	}

	return score
}

func findWinningMove(groups []game.CardGroup, handCount int) (game.CardGroup, bool) {
	var winning []game.CardGroup
	for _, group := range groups {
		if len(group.Cards) == handCount {
			winning = append(winning, group)
		}
	}
	if len(winning) == 0 {
		return game.CardGroup{}, false
	}
	return chooseSmallestResponse(winning), true
}

func chooseBestLead(groups []game.CardGroup) game.CardGroup {
	sorted := append([]game.CardGroup(nil), groups...)
	sort.Slice(sorted, func(i, j int) bool {
		leftPriority := leadGroupPriority(sorted[i].Type)
		rightPriority := leadGroupPriority(sorted[j].Type)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if sorted[i].PrimaryRank != sorted[j].PrimaryRank {
			return sorted[i].PrimaryRank < sorted[j].PrimaryRank
		}
		if sorted[i].Length != sorted[j].Length {
			return sorted[i].Length < sorted[j].Length
		}
		if len(sorted[i].Cards) != len(sorted[j].Cards) {
			return len(sorted[i].Cards) < len(sorted[j].Cards)
		}
		return groupCardsKey(sorted[i]) < groupCardsKey(sorted[j])
	})
	return sorted[0]
}

func chooseSmallestResponse(groups []game.CardGroup) game.CardGroup {
	sorted := append([]game.CardGroup(nil), groups...)
	sort.Slice(sorted, func(i, j int) bool {
		if responseGroupPriority(sorted[i].Type) != responseGroupPriority(sorted[j].Type) {
			return responseGroupPriority(sorted[i].Type) < responseGroupPriority(sorted[j].Type)
		}
		if sorted[i].PrimaryRank != sorted[j].PrimaryRank {
			return sorted[i].PrimaryRank < sorted[j].PrimaryRank
		}
		if sorted[i].Length != sorted[j].Length {
			return sorted[i].Length < sorted[j].Length
		}
		if len(sorted[i].Cards) != len(sorted[j].Cards) {
			return len(sorted[i].Cards) < len(sorted[j].Cards)
		}
		return groupCardsKey(sorted[i]) < groupCardsKey(sorted[j])
	})
	return sorted[0]
}

func findLowestGroup(groups []game.CardGroup, groupType game.CardGroupType) (game.CardGroup, bool) {
	matched := filterGroups(groups, func(group game.CardGroup) bool {
		return group.Type == groupType
	})
	if len(matched) == 0 {
		return game.CardGroup{}, false
	}
	return chooseSmallestResponse(matched), true
}

func filterGroups(groups []game.CardGroup, keep func(game.CardGroup) bool) []game.CardGroup {
	filtered := make([]game.CardGroup, 0, len(groups))
	for _, group := range groups {
		if keep(group) {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func isBombOrRocket(group game.CardGroup) bool {
	return group.Type == game.CardGroupTypeBomb || group.Type == game.CardGroupTypeRocket
}

func isFarmerTeammatePlay(currentGame *game.Game, seatIndex int, previousSeatIndex int) bool {
	if previousSeatIndex < 0 || previousSeatIndex >= len(currentGame.Players) {
		return false
	}
	return currentGame.Players[seatIndex].Role == game.RoleFarmer &&
		currentGame.Players[previousSeatIndex].Role == game.RoleFarmer
}

func hasOpponentWithFewCards(currentGame *game.Game, seatIndex int, maxRemaining int) bool {
	role := currentGame.Players[seatIndex].Role
	for _, player := range currentGame.Players {
		if player.SeatIndex == seatIndex || player.Role == role {
			continue
		}
		if player.RemainingCount > 0 && player.RemainingCount <= maxRemaining {
			return true
		}
	}
	return false
}

func leadGroupPriority(groupType game.CardGroupType) int {
	switch groupType {
	case game.CardGroupTypeStraight:
		return 1
	case game.CardGroupTypePairStraight:
		return 2
	case game.CardGroupTypeAirplane:
		return 3
	case game.CardGroupTypeAirplaneWithSingles, game.CardGroupTypeAirplaneWithPairs:
		return 4
	case game.CardGroupTypeTripleWithSingle, game.CardGroupTypeTripleWithPair:
		return 5
	case game.CardGroupTypeTriple:
		return 6
	case game.CardGroupTypePair:
		return 7
	case game.CardGroupTypeSingle:
		return 8
	case game.CardGroupTypeFourWithTwoSingles, game.CardGroupTypeFourWithTwoPairs:
		return 9
	case game.CardGroupTypeBomb:
		return 98
	case game.CardGroupTypeRocket:
		return 99
	default:
		return 100
	}
}

func responseGroupPriority(groupType game.CardGroupType) int {
	switch groupType {
	case game.CardGroupTypeRocket:
		return 99
	case game.CardGroupTypeBomb:
		return 98
	default:
		return 1
	}
}

func groupCardsKey(group game.CardGroup) string {
	codes := make([]string, 0, len(group.Cards))
	for _, card := range group.Cards {
		codes = append(codes, card.Code())
	}
	sort.Strings(codes)
	return strings.Join(codes, ",")
}
