package app

import (
	"ddz/backend/internal/record"
	"ddz/backend/internal/storage/redisstore"
)

// NewRedisStore 根据配置创建当前使用的短期状态存储。
// 当前默认返回内存实现，后续接入真实 Redis 时保留相同接口边界。
func NewRedisStore(_ Config) redisstore.Store {
	return redisstore.NewMemoryStore()
}

// NewRecordStore 根据配置创建对局记录存储。
// 当前默认使用内存实现，后续可替换为数据库持久化实现。
func NewRecordStore(_ Config, _ redisstore.Store) record.Store {
	return record.NewMemoryStore()
}
