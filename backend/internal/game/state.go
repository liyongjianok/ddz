package game

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidGameSetup = errors.New("invalid game setup")

type GamePhase string

const (
	GamePhaseCreated GamePhase = "created"
	GamePhaseDealing GamePhase = "dealing"
	GamePhaseBidding GamePhase = "bidding"
	GamePhasePlaying GamePhase = "playing"
	GamePhaseEnded   GamePhase = "ended"
	GamePhaseAborted GamePhase = "aborted"
)

type Role string

const (
	RoleNone     Role = ""
	RoleLandlord Role = "landlord"
	RoleFarmer   Role = "farmer"
)

type PlayerStatus string

const (
	PlayerStatusJoined  PlayerStatus = "joined"
	PlayerStatusReady   PlayerStatus = "ready"
	PlayerStatusPlaying PlayerStatus = "playing"
	PlayerStatusOffline PlayerStatus = "offline"
	PlayerStatusLeft    PlayerStatus = "left"
)

// GamePlayerInput 描述开局时传入的玩家座位信息。
type GamePlayerInput struct {
	UserID    string
	SeatIndex int
	IsRobot   bool
}

// Bid 表示一次叫分记录；具体流转在后续 T-2002 实现。
type Bid struct {
	SeatIndex int
	UserID    string
	Score     int
	CreatedAt time.Time
}

// Play 表示一次出牌记录；具体流转在后续 T-2003 实现。
type Play struct {
	SeatIndex int
	UserID    string
	Cards     []Card
	Group     CardGroup
	CreatedAt time.Time
}

// PlayerState 保存对局内玩家的动态信息。
type PlayerState struct {
	UserID         string
	SeatIndex      int
	Role           Role
	Hand           []Card
	Status         PlayerStatus
	IsRobot        bool
	BidScore       int
	RemainingCount int
}

// BiddingState 保存叫分阶段的起始和当前轮次信息。
type BiddingState struct {
	StartSeatIndex      int
	CurrentSeatIndex    int
	HighestBid          int
	HighestBidSeatIndex int
	Bids                []Bid
	Rounds              int
}

// Game 表示一局斗地主的完整领域状态。
type Game struct {
	ID                string
	Phase             GamePhase
	Deck              []Card
	BottomCards       []Card
	Players           []PlayerState
	LandlordSeatIndex int
	CurrentSeatIndex  int
	LastPlay          *Play
	PassCount         int
	BiddingState      BiddingState
	Multiplier        int
	EventSeq          int
	StartedAt         time.Time
	EndedAt           time.Time
}

// NewGame 创建一局新的斗地主，并完成洗牌、发牌和叫分起始座位初始化。
func NewGame(gameID string, players []GamePlayerInput, rng RNG) (*Game, error) {
	if err := validateGameSetup(gameID, players); err != nil {
		return nil, err
	}

	game := &Game{
		ID:                gameID,
		Phase:             GamePhaseCreated,
		LandlordSeatIndex: -1,
		CurrentSeatIndex:  -1,
		Multiplier:        1,
		StartedAt:         time.Now().UTC(),
	}

	// 先构造玩家骨架，保证发牌后手牌按 seat_index 稳定落位。
	game.Players = make([]PlayerState, PlayerCount)
	for _, input := range players {
		game.Players[input.SeatIndex] = PlayerState{
			UserID:    input.UserID,
			SeatIndex: input.SeatIndex,
			Role:      RoleNone,
			Status:    PlayerStatusPlaying,
			IsRobot:   input.IsRobot,
		}
	}

	game.Phase = GamePhaseDealing
	deck := NewDeck()
	deck = Shuffle(deck, rng)
	hands, bottom, err := Deal(deck)
	if err != nil {
		return nil, err
	}

	game.Deck = deck
	game.BottomCards = bottom
	for seatIndex := range game.Players {
		game.Players[seatIndex].Hand = hands[seatIndex]
		game.Players[seatIndex].RemainingCount = len(hands[seatIndex])
	}

	startSeat := 0
	if rng != nil {
		startSeat = rng.Intn(PlayerCount)
	}

	game.Phase = GamePhaseBidding
	game.CurrentSeatIndex = startSeat
	game.BiddingState = BiddingState{
		StartSeatIndex:      startSeat,
		CurrentSeatIndex:    startSeat,
		HighestBid:          0,
		HighestBidSeatIndex: -1,
		Bids:                nil,
		Rounds:              0,
	}

	return game, nil
}

func validateGameSetup(gameID string, players []GamePlayerInput) error {
	if gameID == "" {
		return fmt.Errorf("%w: game id is empty", ErrInvalidGameSetup)
	}
	if len(players) != PlayerCount {
		return fmt.Errorf("%w: expected %d players, got %d", ErrInvalidGameSetup, PlayerCount, len(players))
	}

	seenSeats := make(map[int]struct{}, len(players))
	seenUsers := make(map[string]struct{}, len(players))
	for _, player := range players {
		if player.UserID == "" {
			return fmt.Errorf("%w: user id is empty", ErrInvalidGameSetup)
		}
		if player.SeatIndex < 0 || player.SeatIndex >= PlayerCount {
			return fmt.Errorf("%w: invalid seat index %d", ErrInvalidGameSetup, player.SeatIndex)
		}
		if _, exists := seenSeats[player.SeatIndex]; exists {
			return fmt.Errorf("%w: duplicate seat index %d", ErrInvalidGameSetup, player.SeatIndex)
		}
		if _, exists := seenUsers[player.UserID]; exists {
			return fmt.Errorf("%w: duplicate user id %s", ErrInvalidGameSetup, player.UserID)
		}
		seenSeats[player.SeatIndex] = struct{}{}
		seenUsers[player.UserID] = struct{}{}
	}

	return nil
}
