package room

import (
	"sort"
	"time"

	"ddz/backend/internal/game"
)

// RoomSnapshot 表示面向单个玩家的房间快照。
type RoomSnapshot struct {
	Room    RoomSnapshotRoom     `json:"room"`
	Players []RoomSnapshotPlayer `json:"players"`
	Game    *RoomSnapshotGame    `json:"game,omitempty"`
	Me      RoomSnapshotMe       `json:"me"`
}

// RoomSnapshotRoom 表示快照中的公开房间信息。
type RoomSnapshotRoom struct {
	RoomID    string `json:"room_id"`
	Mode      string `json:"mode"`
	Status    string `json:"status"`
	BaseScore int    `json:"base_score"`
}

// RoomSnapshotPlayer 表示快照中的公开玩家信息。
type RoomSnapshotPlayer struct {
	UserID         string `json:"user_id"`
	SeatIndex      int    `json:"seat_index"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	Ready          bool   `json:"ready"`
	RemainingCount int    `json:"remaining_count"`
	IsRobot        bool   `json:"is_robot"`
}

// RoomSnapshotGame 表示快照中的公开游戏状态。
type RoomSnapshotGame struct {
	GameID            string                  `json:"game_id"`
	Phase             string                  `json:"phase"`
	CurrentSeatIndex  int                     `json:"current_seat_index"`
	LandlordSeatIndex int                     `json:"landlord_seat_index"`
	BottomCards       []string                `json:"bottom_cards,omitempty"`
	LastPlay          *RoomSnapshotPlay       `json:"last_play,omitempty"`
	Multiplier        int                     `json:"multiplier"`
	Settlement        *RoomSnapshotSettlement `json:"settlement,omitempty"`
}

// RoomSnapshotPlay 表示上一手出牌的公开信息。
type RoomSnapshotPlay struct {
	SeatIndex int                   `json:"seat_index"`
	UserID    string                `json:"user_id"`
	Cards     []string              `json:"cards"`
	Group     RoomSnapshotCardGroup `json:"card_group"`
	CreatedAt time.Time             `json:"created_at"`
}

// RoomSnapshotCardGroup 表示出牌牌型信息。
type RoomSnapshotCardGroup struct {
	Type        string   `json:"type"`
	Rank        string   `json:"rank"`
	Length      int      `json:"length"`
	Attachments []string `json:"attachments,omitempty"`
}

// RoomSnapshotSettlement 表示对局结束后的结算信息。
type RoomSnapshotSettlement struct {
	WinnerSide      string                         `json:"winner_side"`
	FinalMultiplier int                            `json:"final_multiplier"`
	BaseScore       int                            `json:"base_score"`
	Players         []RoomSnapshotSettlementPlayer `json:"players"`
}

// RoomSnapshotSettlementPlayer 表示单个玩家的结算结果。
type RoomSnapshotSettlementPlayer struct {
	UserID     string `json:"user_id"`
	SeatIndex  int    `json:"seat_index"`
	Role       string `json:"role"`
	ScoreDelta int    `json:"score_delta"`
	IsWinner   bool   `json:"is_winner"`
}

// RoomSnapshotMe 表示当前玩家的私有快照。
type RoomSnapshotMe struct {
	UserID    string   `json:"user_id"`
	SeatIndex int      `json:"seat_index"`
	Hand      []string `json:"hand,omitempty"`
}

// BuildRoomSnapshot 根据房间当前状态构建单个玩家的专属快照。
func BuildRoomSnapshot(room *Room, userID string) (*RoomSnapshot, error) {
	if room == nil {
		return nil, ErrRoomNotFound
	}

	seat, ok := findSeatByUserID(room.Seats, userID)
	if !ok {
		return nil, ErrUserNotInRoom
	}

	snapshot := &RoomSnapshot{
		Room: RoomSnapshotRoom{
			RoomID:    room.ID,
			Mode:      room.Mode,
			Status:    string(room.Status),
			BaseScore: room.BaseScore,
		},
		Players: buildSnapshotPlayers(room),
		Me: RoomSnapshotMe{
			UserID:    seat.UserID,
			SeatIndex: seat.SeatIndex,
		},
	}

	if room.CurrentGame == nil {
		return snapshot, nil
	}

	mePlayer, ok := findGamePlayerByUserID(room.CurrentGame.Players, userID)
	if !ok {
		return nil, ErrUserNotInRoom
	}
	snapshot.Me.Hand = cardsToCodesSnapshot(mePlayer.Hand)

	gameSnapshot, err := buildGameSnapshot(room)
	if err != nil {
		return nil, err
	}
	snapshot.Game = gameSnapshot
	return snapshot, nil
}

func buildSnapshotPlayers(room *Room) []RoomSnapshotPlayer {
	seats := append([]Seat(nil), room.Seats...)
	sort.Slice(seats, func(i, j int) bool {
		return seats[i].SeatIndex < seats[j].SeatIndex
	})

	players := make([]RoomSnapshotPlayer, 0, len(seats))
	playerBySeat := make(map[int]game.PlayerState, len(seats))
	if room.CurrentGame != nil {
		for _, player := range room.CurrentGame.Players {
			playerBySeat[player.SeatIndex] = player
		}
	}

	for _, seat := range seats {
		playerStatus := game.PlayerStatusJoined
		role := game.RoleNone
		remainingCount := 0

		if room.CurrentGame != nil {
			if player, exists := playerBySeat[seat.SeatIndex]; exists {
				playerStatus = player.Status
				role = player.Role
				remainingCount = player.RemainingCount
			}
		} else if seat.Ready {
			playerStatus = game.PlayerStatusReady
		}

		players = append(players, RoomSnapshotPlayer{
			UserID:         seat.UserID,
			SeatIndex:      seat.SeatIndex,
			Role:           string(role),
			Status:         string(playerStatus),
			Ready:          seat.Ready,
			RemainingCount: remainingCount,
			IsRobot:        seat.IsRobot,
		})
	}

	return players
}

func buildGameSnapshot(room *Room) (*RoomSnapshotGame, error) {
	g := room.CurrentGame
	if g == nil {
		return nil, nil
	}

	snapshot := &RoomSnapshotGame{
		GameID:            g.ID,
		Phase:             string(g.Phase),
		CurrentSeatIndex:  g.CurrentSeatIndex,
		LandlordSeatIndex: g.LandlordSeatIndex,
		Multiplier:        g.Multiplier,
	}

	if g.LandlordSeatIndex >= 0 {
		snapshot.BottomCards = cardsToCodesSnapshot(g.BottomCards)
	}
	if g.LastPlay != nil {
		snapshot.LastPlay = buildPlaySnapshot(g.LastPlay)
	}
	if g.Phase == game.GamePhaseEnded {
		settlement := g.Settlement
		if settlement == nil {
			var err error
			settlement, err = g.Settle(room.BaseScore)
			if err != nil {
				return nil, err
			}
		}
		snapshot.Settlement = buildSettlementSnapshot(settlement)
	}

	return snapshot, nil
}

func buildPlaySnapshot(play *game.Play) *RoomSnapshotPlay {
	if play == nil {
		return nil
	}

	return &RoomSnapshotPlay{
		SeatIndex: play.SeatIndex,
		UserID:    play.UserID,
		Cards:     cardsToCodesSnapshot(play.Cards),
		Group: RoomSnapshotCardGroup{
			Type:        string(play.Group.Type),
			Rank:        play.Group.PrimaryRank.String(),
			Length:      play.Group.Length,
			Attachments: cardsToCodesSnapshot(play.Group.Attachments),
		},
		CreatedAt: play.CreatedAt,
	}
}

func buildSettlementSnapshot(settlement *game.SettlementResult) *RoomSnapshotSettlement {
	if settlement == nil {
		return nil
	}

	players := make([]RoomSnapshotSettlementPlayer, 0, len(settlement.Players))
	for _, player := range settlement.Players {
		players = append(players, RoomSnapshotSettlementPlayer{
			UserID:     player.UserID,
			SeatIndex:  player.SeatIndex,
			Role:       string(player.Role),
			ScoreDelta: player.DeltaScore,
			IsWinner:   player.IsWinner,
		})
	}

	return &RoomSnapshotSettlement{
		WinnerSide:      string(settlement.WinnerSide),
		FinalMultiplier: settlement.Multiplier,
		BaseScore:       settlement.BaseScore,
		Players:         players,
	}
}

func cardsToCodesSnapshot(cards []game.Card) []string {
	if len(cards) == 0 {
		return nil
	}

	result := make([]string, 0, len(cards))
	for _, card := range cards {
		result = append(result, card.Code())
	}
	return result
}

func findSeatByUserID(seats []Seat, userID string) (Seat, bool) {
	for _, seat := range seats {
		if seat.UserID == userID {
			return seat, true
		}
	}
	return Seat{}, false
}

func findGamePlayerByUserID(players []game.PlayerState, userID string) (game.PlayerState, bool) {
	for _, player := range players {
		if player.UserID == userID {
			return player, true
		}
	}
	return game.PlayerState{}, false
}
