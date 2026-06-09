package record

import (
	"context"
	"errors"
	"time"

	"ddz/backend/internal/game"
	"ddz/backend/internal/room"
)

var ErrRecordNotFound = errors.New("record not found")
var ErrRecordForbidden = errors.New("record forbidden")

// Service 封装对局记录的写入与查询逻辑。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建记录服务。
func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// AppendEvent 追加一条事件。
func (s *Service) AppendEvent(ctx context.Context, event Event) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.now()
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	return s.store.AppendEvent(ctx, event)
}

// SaveGameRecord 保存完整对局记录。
func (s *Service) SaveGameRecord(ctx context.Context, currentRoom *room.Room) error {
	if currentRoom == nil || currentRoom.CurrentGame == nil || currentRoom.CurrentGame.Settlement == nil {
		return nil
	}

	record := GameRecord{
		GameID:       currentRoom.CurrentGame.ID,
		RoomID:       currentRoom.ID,
		Mode:         currentRoom.Mode,
		BaseScore:    currentRoom.BaseScore,
		Multiplier:   currentRoom.CurrentGame.Settlement.Multiplier,
		WinnerSide:   string(currentRoom.CurrentGame.Settlement.WinnerSide),
		StartedAt:    currentRoom.CurrentGame.StartedAt,
		EndedAt:      currentRoom.CurrentGame.EndedAt,
		Participants: make([]string, 0, len(currentRoom.CurrentGame.Players)),
		Players:      make([]RecordPlayer, 0, len(currentRoom.CurrentGame.Settlement.Players)),
	}

	for _, player := range currentRoom.CurrentGame.Players {
		record.Participants = append(record.Participants, player.UserID)
	}

	for _, player := range currentRoom.CurrentGame.Settlement.Players {
		record.Players = append(record.Players, RecordPlayer{
			UserID:      player.UserID,
			DisplayName: player.UserID,
			SeatIndex:   player.SeatIndex,
			Role:        string(player.Role),
			ScoreDelta:  player.DeltaScore,
		})
	}

	return s.store.SaveGameRecord(ctx, record)
}

// ListMyRecords 查询当前用户的对局记录列表。
func (s *Service) ListMyRecords(ctx context.Context, userID string, page int, pageSize int) (RecordList, error) {
	return s.store.ListRecordsByUser(ctx, userID, page, pageSize)
}

// GetGameRecord 查询指定对局记录，仅允许参与者查看。
func (s *Service) GetGameRecord(ctx context.Context, userID string, gameID string) (GameRecord, error) {
	record, ok, err := s.store.GetGameRecord(ctx, gameID)
	if err != nil {
		return GameRecord{}, err
	}
	if !ok {
		return GameRecord{}, ErrRecordNotFound
	}
	for _, participant := range record.Participants {
		if participant == userID {
			return record, nil
		}
	}
	return GameRecord{}, ErrRecordForbidden
}

// BuildEventFromBid 构建叫分事件。
func BuildEventFromBid(currentRoom *room.Room, actorUserID string, seatIndex int, score int) Event {
	return Event{
		GameID:      currentRoom.CurrentGame.ID,
		RoomID:      currentRoom.ID,
		Seq:         nextEventSeq(currentRoom.CurrentGame),
		EventType:   "bid_placed",
		ActorUserID: actorUserID,
		Payload: map[string]any{
			"seat_index": seatIndex,
			"score":      score,
		},
	}
}

// BuildEventFromPlay 构建出牌事件。
func BuildEventFromPlay(currentRoom *room.Room, actorUserID string, lastPlay *game.Play, remainingCount int) Event {
	payload := map[string]any{
		"seat_index":      lastPlay.SeatIndex,
		"cards":           cardCodes(lastPlay.Cards),
		"remaining_count": remainingCount,
		"card_group": map[string]any{
			"type":   string(lastPlay.Group.Type),
			"rank":   lastPlay.Group.PrimaryRank.String(),
			"length": lastPlay.Group.Length,
		},
	}
	if attachments := cardCodes(lastPlay.Group.Attachments); len(attachments) > 0 {
		payload["card_group"].(map[string]any)["attachments"] = attachments
	}

	return Event{
		GameID:      currentRoom.CurrentGame.ID,
		RoomID:      currentRoom.ID,
		Seq:         nextEventSeq(currentRoom.CurrentGame),
		EventType:   "cards_played",
		ActorUserID: actorUserID,
		Payload:     payload,
	}
}

// BuildEventFromPass 构建不出事件。
func BuildEventFromPass(currentRoom *room.Room, actorUserID string, seatIndex int) Event {
	return Event{
		GameID:      currentRoom.CurrentGame.ID,
		RoomID:      currentRoom.ID,
		Seq:         nextEventSeq(currentRoom.CurrentGame),
		EventType:   "player_passed",
		ActorUserID: actorUserID,
		Payload: map[string]any{
			"seat_index": seatIndex,
		},
	}
}

// BuildEventFromSettlement 构建对局结束事件。
func BuildEventFromSettlement(currentRoom *room.Room) Event {
	return Event{
		GameID:    currentRoom.CurrentGame.ID,
		RoomID:    currentRoom.ID,
		Seq:       nextEventSeq(currentRoom.CurrentGame),
		EventType: "game_ended",
		Payload: map[string]any{
			"winner_side":      string(currentRoom.CurrentGame.Settlement.WinnerSide),
			"final_multiplier": currentRoom.CurrentGame.Settlement.Multiplier,
		},
	}
}

func nextEventSeq(currentGame *game.Game) int {
	currentGame.EventSeq++
	return currentGame.EventSeq
}

func cardCodes(cards []game.Card) []string {
	result := make([]string, 0, len(cards))
	for _, card := range cards {
		result = append(result, card.Code())
	}
	return result
}
