package game

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidGameSetup = errors.New("invalid game setup")
var ErrInvalidGamePhase = errors.New("invalid game phase")
var ErrNotPlayerTurn = errors.New("not player turn")
var ErrInvalidBid = errors.New("invalid bid")
var ErrCannotPass = errors.New("cannot pass")

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
	RedealCount         int
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

	startSeat := 0
	if rng != nil {
		startSeat = rng.Intn(PlayerCount)
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

	game.Phase = GamePhaseBidding
	game.CurrentSeatIndex = startSeat
	game.BiddingState = BiddingState{
		StartSeatIndex:      startSeat,
		CurrentSeatIndex:    startSeat,
		HighestBid:          0,
		HighestBidSeatIndex: -1,
		Bids:                nil,
		Rounds:              0,
		RedealCount:         0,
	}

	return game, nil
}

// PlaceBid 处理叫分动作，并在条件满足时推进到地主确定或重发牌。
func (g *Game) PlaceBid(seatIndex int, score int, rng RNG) error {
	if g.Phase != GamePhaseBidding {
		return ErrInvalidGamePhase
	}
	if seatIndex != g.BiddingState.CurrentSeatIndex || seatIndex != g.CurrentSeatIndex {
		return ErrNotPlayerTurn
	}
	if score < 0 || score > 3 {
		return ErrInvalidBid
	}
	if score != 0 && score <= g.BiddingState.HighestBid {
		return ErrInvalidBid
	}

	player := &g.Players[seatIndex]
	bid := Bid{
		SeatIndex: seatIndex,
		UserID:    player.UserID,
		Score:     score,
		CreatedAt: time.Now().UTC(),
	}

	g.BiddingState.Bids = append(g.BiddingState.Bids, bid)
	g.BiddingState.Rounds++
	player.BidScore = score

	if score > g.BiddingState.HighestBid {
		g.BiddingState.HighestBid = score
		g.BiddingState.HighestBidSeatIndex = seatIndex
	}

	if score == 3 {
		g.assignLandlord(seatIndex)
		return nil
	}

	if len(g.BiddingState.Bids) >= PlayerCount {
		if g.BiddingState.HighestBid > 0 {
			g.assignLandlord(g.BiddingState.HighestBidSeatIndex)
			return nil
		}
		return g.handleAllPass(rng)
	}

	nextSeat := (seatIndex + 1) % PlayerCount
	g.BiddingState.CurrentSeatIndex = nextSeat
	g.CurrentSeatIndex = nextSeat
	return nil
}

// PlayCards 处理当前玩家的出牌动作，并在必要时推进到下一轮或结束游戏。
func (g *Game) PlayCards(seatIndex int, cards []Card) error {
	if g.Phase != GamePhasePlaying {
		return ErrInvalidGamePhase
	}
	if seatIndex != g.CurrentSeatIndex {
		return ErrNotPlayerTurn
	}
	if len(cards) == 0 {
		return ErrInvalidCardSet
	}

	player := &g.Players[seatIndex]
	if !ContainsCards(player.Hand, cards) {
		return ErrInvalidCardSet
	}

	group, err := Recognize(cards)
	if err != nil {
		return err
	}
	if g.LastPlay != nil && !CanBeat(group, g.LastPlay.Group) {
		return ErrInvalidCardSet
	}

	remaining, err := RemoveCards(player.Hand, cards)
	if err != nil {
		return err
	}

	player.Hand = remaining
	player.RemainingCount = len(remaining)
	g.LastPlay = &Play{
		SeatIndex: seatIndex,
		UserID:    player.UserID,
		Cards:     copyCards(cards),
		Group:     group,
		CreatedAt: time.Now().UTC(),
	}
	g.PassCount = 0

	if len(remaining) == 0 {
		g.Phase = GamePhaseEnded
		g.EndedAt = time.Now().UTC()
		g.CurrentSeatIndex = seatIndex
		return nil
	}

	g.CurrentSeatIndex = nextSeatIndex(seatIndex)
	return nil
}

// Pass 处理“不出”动作；当连续两家不出时，上一手出牌者重新获得主动权。
func (g *Game) Pass(seatIndex int) error {
	if g.Phase != GamePhasePlaying {
		return ErrInvalidGamePhase
	}
	if seatIndex != g.CurrentSeatIndex {
		return ErrNotPlayerTurn
	}
	if g.LastPlay == nil {
		return ErrCannotPass
	}

	g.PassCount++
	if g.PassCount >= 2 {
		leadSeat := g.LastPlay.SeatIndex
		g.PassCount = 0
		g.LastPlay = nil
		g.CurrentSeatIndex = leadSeat
		return nil
	}

	g.CurrentSeatIndex = nextSeatIndex(seatIndex)
	return nil
}

// assignLandlord 在叫分结束后设置地主、分配底牌并切到出牌阶段。
func (g *Game) assignLandlord(seatIndex int) {
	g.LandlordSeatIndex = seatIndex
	g.CurrentSeatIndex = seatIndex
	g.BiddingState.CurrentSeatIndex = seatIndex
	g.Phase = GamePhasePlaying
	if g.BiddingState.HighestBid > 0 {
		g.Multiplier = g.BiddingState.HighestBid
	} else {
		g.Multiplier = 1
	}

	for i := range g.Players {
		g.Players[i].Role = RoleFarmer
	}

	landlord := &g.Players[seatIndex]
	landlord.Role = RoleLandlord
	landlord.Hand = append(landlord.Hand, g.BottomCards...)
	landlord.RemainingCount = len(landlord.Hand)

	for i := range g.Players {
		if i == seatIndex {
			continue
		}
		g.Players[i].RemainingCount = len(g.Players[i].Hand)
	}
}

// handleAllPass 处理三人都不叫的场景：先重发一次，再次全不叫则随机指定地主。
func (g *Game) handleAllPass(rng RNG) error {
	if g.BiddingState.RedealCount == 0 {
		return g.redealForAllPass(rng)
	}

	seatIndex := 0
	if rng != nil {
		seatIndex = rng.Intn(PlayerCount)
	}
	g.BiddingState.HighestBid = 1
	g.BiddingState.HighestBidSeatIndex = seatIndex
	g.Players[seatIndex].BidScore = 1
	g.assignLandlord(seatIndex)
	return nil
}

// redealForAllPass 在首次全部不叫时重新洗牌发牌，并重置叫分状态。
func (g *Game) redealForAllPass(rng RNG) error {
	startSeat := 0
	if rng != nil {
		startSeat = rng.Intn(PlayerCount)
	}

	deck := Shuffle(NewDeck(), rng)
	hands, bottom, err := Deal(deck)
	if err != nil {
		return err
	}

	g.Deck = deck
	g.BottomCards = bottom
	for seatIndex := range g.Players {
		g.Players[seatIndex].Hand = hands[seatIndex]
		g.Players[seatIndex].RemainingCount = len(hands[seatIndex])
		g.Players[seatIndex].BidScore = 0
		g.Players[seatIndex].Role = RoleNone
	}

	g.Phase = GamePhaseBidding
	g.CurrentSeatIndex = startSeat
	g.LandlordSeatIndex = -1
	g.Multiplier = 1
	g.BiddingState = BiddingState{
		StartSeatIndex:      startSeat,
		CurrentSeatIndex:    startSeat,
		HighestBid:          0,
		HighestBidSeatIndex: -1,
		Bids:                nil,
		Rounds:              0,
		RedealCount:         1,
	}

	return nil
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

func nextSeatIndex(seatIndex int) int {
	return (seatIndex + 1) % PlayerCount
}
