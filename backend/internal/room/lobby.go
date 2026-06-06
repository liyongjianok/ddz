package room

import (
	"fmt"
	"sort"
	"time"
)

const (
	defaultRoomListPage     = 1
	defaultRoomListPageSize = 20
	maxRoomListPageSize     = 100
)

// LobbySummary 表示大厅页需要展示的聚合统计数据。
type LobbySummary struct {
	OnlinePlayers int                `json:"online_players"`
	ActiveRooms   int                `json:"active_rooms"`
	Modes         []LobbyModeSummary `json:"modes"`
}

// LobbyModeSummary 表示单个玩法和底分组合的大厅统计。
type LobbyModeSummary struct {
	Mode          string `json:"mode"`
	BaseScore     int    `json:"base_score"`
	OnlinePlayers int    `json:"online_players"`
	WaitingRooms  int    `json:"waiting_rooms"`
}

// RoomListFilter 表示房间列表查询的过滤和分页参数。
type RoomListFilter struct {
	Mode     string
	Status   RoomStatus
	Page     int
	PageSize int
}

// RoomListItem 表示房间列表中的公开房间摘要。
type RoomListItem struct {
	RoomID      string    `json:"room_id"`
	Mode        string    `json:"mode"`
	Status      string    `json:"status"`
	BaseScore   int       `json:"base_score"`
	PlayerCount int       `json:"player_count"`
	MaxPlayers  int       `json:"max_players"`
	CreatedAt   time.Time `json:"created_at"`
}

// RoomListResult 表示分页后的房间列表结果。
type RoomListResult struct {
	Items    []RoomListItem `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int            `json:"total"`
}

// GetLobbySummary 统计当前所有活跃房间的大厅聚合信息。
func (m *Manager) GetLobbySummary() (LobbySummary, error) {
	rooms, err := m.listRoomSnapshots()
	if err != nil {
		return LobbySummary{}, err
	}

	summary := LobbySummary{
		Modes: make([]LobbyModeSummary, 0),
	}
	modeStats := make(map[string]*LobbyModeSummary)

	for _, currentRoom := range rooms {
		if currentRoom.Status == RoomStatusClosed {
			continue
		}

		playerCount := len(currentRoom.Seats)
		summary.OnlinePlayers += playerCount
		summary.ActiveRooms++

		key := lobbyModeKey(currentRoom.Mode, currentRoom.BaseScore)
		if _, exists := modeStats[key]; !exists {
			modeStats[key] = &LobbyModeSummary{
				Mode:      currentRoom.Mode,
				BaseScore: currentRoom.BaseScore,
			}
		}

		modeStats[key].OnlinePlayers += playerCount
		if currentRoom.Status == RoomStatusWaiting {
			modeStats[key].WaitingRooms++
		}
	}

	for _, item := range modeStats {
		summary.Modes = append(summary.Modes, *item)
	}
	sort.Slice(summary.Modes, func(i, j int) bool {
		if summary.Modes[i].Mode != summary.Modes[j].Mode {
			return summary.Modes[i].Mode < summary.Modes[j].Mode
		}
		return summary.Modes[i].BaseScore < summary.Modes[j].BaseScore
	})

	return summary, nil
}

// ListRooms 返回按条件筛选后的房间公开列表，不包含任何隐藏牌信息。
func (m *Manager) ListRooms(filter RoomListFilter) (RoomListResult, error) {
	page := normalizeRoomListPage(filter.Page)
	pageSize := normalizeRoomListPageSize(filter.PageSize)

	rooms, err := m.listRoomSnapshots()
	if err != nil {
		return RoomListResult{}, err
	}

	items := make([]RoomListItem, 0, len(rooms))
	for _, currentRoom := range rooms {
		if currentRoom.Status == RoomStatusClosed {
			continue
		}
		if filter.Mode != "" && currentRoom.Mode != filter.Mode {
			continue
		}
		if filter.Status != "" && currentRoom.Status != filter.Status {
			continue
		}

		items = append(items, RoomListItem{
			RoomID:      currentRoom.ID,
			Mode:        currentRoom.Mode,
			Status:      string(currentRoom.Status),
			BaseScore:   currentRoom.BaseScore,
			PlayerCount: len(currentRoom.Seats),
			MaxPlayers:  currentRoom.MaxPlayers,
			CreatedAt:   currentRoom.CreatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].RoomID > items[j].RoomID
	})

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return RoomListResult{
		Items:    items[start:end],
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func (m *Manager) listRoomSnapshots() ([]Room, error) {
	m.mu.RLock()
	actors := make([]*RoomActor, 0, len(m.rooms))
	for _, actor := range m.rooms {
		actors = append(actors, actor)
	}
	m.mu.RUnlock()

	rooms := make([]Room, 0, len(actors))
	for _, actor := range actors {
		currentRoom, err := actor.Snapshot()
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, currentRoom)
	}
	return rooms, nil
}

func normalizeRoomListPage(page int) int {
	if page < 1 {
		return defaultRoomListPage
	}
	return page
}

func normalizeRoomListPageSize(pageSize int) int {
	if pageSize < 1 {
		return defaultRoomListPageSize
	}
	if pageSize > maxRoomListPageSize {
		return maxRoomListPageSize
	}
	return pageSize
}

func lobbyModeKey(mode string, baseScore int) string {
	return fmt.Sprintf("%s|%d", mode, baseScore)
}
