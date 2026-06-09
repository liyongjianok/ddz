package record

import "context"

// Store 定义对局记录与事件的持久化边界。
type Store interface {
	AppendEvent(ctx context.Context, event Event) error
	SaveGameRecord(ctx context.Context, record GameRecord) error
	ListRecordsByUser(ctx context.Context, userID string, page int, pageSize int) (RecordList, error)
	GetGameRecord(ctx context.Context, gameID string) (GameRecord, bool, error)
}
