package game

func CanBeat(candidate CardGroup, previous CardGroup) bool {
	if candidate.Type == "" || previous.Type == "" {
		return false
	}

	if previous.Type == CardGroupTypeRocket {
		return false
	}
	if candidate.Type == CardGroupTypeRocket {
		return true
	}

	if candidate.Type == CardGroupTypeBomb {
		if previous.Type != CardGroupTypeBomb {
			return true
		}
		return candidate.PrimaryRank > previous.PrimaryRank
	}
	if previous.Type == CardGroupTypeBomb {
		return false
	}

	if candidate.Type != previous.Type {
		return false
	}
	if !sameStructure(candidate, previous) {
		return false
	}

	return candidate.PrimaryRank > previous.PrimaryRank
}

func sameStructure(candidate CardGroup, previous CardGroup) bool {
	switch candidate.Type {
	case CardGroupTypeStraight,
		CardGroupTypePairStraight,
		CardGroupTypeAirplane,
		CardGroupTypeAirplaneWithSingles,
		CardGroupTypeAirplaneWithPairs:
		return candidate.Length == previous.Length
	default:
		return true
	}
}
