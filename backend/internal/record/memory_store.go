package record

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore 是记录持久化的内存实现。
type MemoryStore struct {
	mu          sync.RWMutex
	games       map[string]GameRecord
	userGameIDs map[string]map[string]struct{}
}

// NewMemoryStore 创建内存版记录存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		games:       make(map[string]GameRecord),
		userGameIDs: make(map[string]map[string]struct{}),
	}
}

func (s *MemoryStore) AppendEvent(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.games[event.GameID]
	record.GameID = event.GameID
	record.RoomID = event.RoomID
	record.Events = append(record.Events, cloneEvents([]Event{event})...)
	s.games[event.GameID] = record
	return nil
}

func (s *MemoryStore) SaveGameRecord(_ context.Context, record GameRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.games[record.GameID]
	record.Events = append(cloneEvents(existing.Events), cloneEvents(record.Events)...)
	record.Participants = append([]string(nil), record.Participants...)
	record.Players = append([]RecordPlayer(nil), record.Players...)
	s.games[record.GameID] = record

	for _, userID := range record.Participants {
		if s.userGameIDs[userID] == nil {
			s.userGameIDs[userID] = make(map[string]struct{})
		}
		s.userGameIDs[userID][record.GameID] = struct{}{}
	}
	return nil
}

func (s *MemoryStore) ListRecordsByUser(_ context.Context, userID string, page int, pageSize int) (RecordList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	gameIDs := s.userGameIDs[userID]
	records := make([]GameRecord, 0, len(gameIDs))
	for gameID := range gameIDs {
		records = append(records, s.games[gameID])
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].EndedAt.After(records[j].EndedAt)
	})

	items := make([]RecordItem, 0, len(records))
	for _, gameRecord := range records {
		item := RecordItem{
			GameID:     gameRecord.GameID,
			Mode:       gameRecord.Mode,
			WinnerSide: gameRecord.WinnerSide,
			StartedAt:  gameRecord.StartedAt,
			EndedAt:    gameRecord.EndedAt,
		}
		for _, player := range gameRecord.Players {
			if player.UserID == userID {
				item.Role = player.Role
				item.ScoreDelta = player.ScoreDelta
				break
			}
		}
		items = append(items, item)
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return RecordList{
			Items:    []RecordItem{},
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		}, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return RecordList{
		Items:    append([]RecordItem(nil), items[start:end]...),
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (s *MemoryStore) GetGameRecord(_ context.Context, gameID string) (GameRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.games[gameID]
	if !ok {
		return GameRecord{}, false, nil
	}

	record.Participants = append([]string(nil), record.Participants...)
	record.Players = append([]RecordPlayer(nil), record.Players...)
	record.Events = cloneEvents(record.Events)
	return record, true, nil
}

func cloneEvents(events []Event) []Event {
	if len(events) == 0 {
		return nil
	}

	result := make([]Event, 0, len(events))
	for _, event := range events {
		cloned := event
		if event.Payload != nil {
			cloned.Payload = make(map[string]any, len(event.Payload))
			for key, value := range event.Payload {
				cloned.Payload[key] = value
			}
		}
		result = append(result, cloned)
	}
	return result
}
