package app

import "ddz/backend/internal/storage/redisstore"

// NewRedisStore 根据配置创建当前使用的短期状态存储。
// 目前默认返回内存实现，后续接入真实 Redis 时保留相同接口边界。
func NewRedisStore(_ Config) redisstore.Store {
	return redisstore.NewMemoryStore()
}
