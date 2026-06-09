package record

import "time"

// Event 表示一条追加写入的游戏事件。
type Event struct {
	GameID      string         `json:"game_id"`
	RoomID      string         `json:"room_id"`
	Seq         int            `json:"seq"`
	EventType   string         `json:"event_type"`
	ActorUserID string         `json:"actor_user_id,omitempty"`
	Payload     map[string]any `json:"payload"`
	CreatedAt   time.Time      `json:"created_at"`
}

// RecordPlayer 表示对局记录中的玩家结果。
type RecordPlayer struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	SeatIndex   int    `json:"seat_index"`
	Role        string `json:"role"`
	ScoreDelta  int    `json:"score_delta"`
}

// RecordItem 表示“我的战绩”列表中的单局摘要。
type RecordItem struct {
	GameID     string    `json:"game_id"`
	Mode       string    `json:"mode"`
	Role       string    `json:"role"`
	WinnerSide string    `json:"winner_side"`
	ScoreDelta int       `json:"score_delta"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

// RecordList 表示分页战绩列表。
type RecordList struct {
	Items    []RecordItem `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int          `json:"total"`
}

// GameRecord 表示一局完整对局记录。
type GameRecord struct {
	GameID       string         `json:"game_id"`
	RoomID       string         `json:"room_id"`
	Mode         string         `json:"mode"`
	BaseScore    int            `json:"base_score"`
	Multiplier   int            `json:"multiplier"`
	WinnerSide   string         `json:"winner_side"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      time.Time      `json:"ended_at"`
	Participants []string       `json:"participants"`
	Players      []RecordPlayer `json:"players"`
	Events       []Event        `json:"events"`
}
