package game

import (
	"errors"
	"fmt"
)

var ErrInvalidCardCode = errors.New("invalid card code")

type Suit string

const (
	SuitNone    Suit = ""
	SuitSpade   Suit = "S"
	SuitHeart   Suit = "H"
	SuitClub    Suit = "C"
	SuitDiamond Suit = "D"
)

var standardSuits = []Suit{
	SuitSpade,
	SuitHeart,
	SuitClub,
	SuitDiamond,
}

type Rank int

const (
	RankInvalid Rank = iota
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
	Rank9
	RankT
	RankJ
	RankQ
	RankK
	RankA
	Rank2
	RankBlackJoker
	RankRedJoker
)

var standardRanks = []Rank{
	Rank3,
	Rank4,
	Rank5,
	Rank6,
	Rank7,
	Rank8,
	Rank9,
	RankT,
	RankJ,
	RankQ,
	RankK,
	RankA,
	Rank2,
}

var rankToCode = map[Rank]string{
	Rank3:          "3",
	Rank4:          "4",
	Rank5:          "5",
	Rank6:          "6",
	Rank7:          "7",
	Rank8:          "8",
	Rank9:          "9",
	RankT:          "T",
	RankJ:          "J",
	RankQ:          "Q",
	RankK:          "K",
	RankA:          "A",
	Rank2:          "2",
	RankBlackJoker: "BJ",
	RankRedJoker:   "RJ",
}

var codeToRank = map[string]Rank{
	"3":  Rank3,
	"4":  Rank4,
	"5":  Rank5,
	"6":  Rank6,
	"7":  Rank7,
	"8":  Rank8,
	"9":  Rank9,
	"T":  RankT,
	"J":  RankJ,
	"Q":  RankQ,
	"K":  RankK,
	"A":  RankA,
	"2":  Rank2,
	"BJ": RankBlackJoker,
	"RJ": RankRedJoker,
}

type Card struct {
	Suit Suit
	Rank Rank
}

func ParseCard(code string) (Card, error) {
	switch len(code) {
	case 2:
		if code == "BJ" || code == "RJ" {
			return Card{
				Suit: SuitNone,
				Rank: codeToRank[code],
			}, nil
		}

		suit, ok := parseSuit(code[:1])
		if !ok {
			return Card{}, fmt.Errorf("%w: %s", ErrInvalidCardCode, code)
		}

		rank, ok := codeToRank[code[1:]]
		if !ok || rank > Rank2 {
			return Card{}, fmt.Errorf("%w: %s", ErrInvalidCardCode, code)
		}

		return Card{
			Suit: suit,
			Rank: rank,
		}, nil
	default:
		return Card{}, fmt.Errorf("%w: %s", ErrInvalidCardCode, code)
	}
}

func (c Card) Code() string {
	if c.IsJoker() {
		return rankToCode[c.Rank]
	}
	return string(c.Suit) + rankToCode[c.Rank]
}

func (c Card) String() string {
	return c.Code()
}

func (c Card) IsJoker() bool {
	return c.Rank == RankBlackJoker || c.Rank == RankRedJoker
}

func (r Rank) String() string {
	if code, ok := rankToCode[r]; ok {
		return code
	}
	return "INVALID"
}

func parseSuit(code string) (Suit, bool) {
	switch Suit(code) {
	case SuitSpade, SuitHeart, SuitClub, SuitDiamond:
		return Suit(code), true
	default:
		return SuitNone, false
	}
}
