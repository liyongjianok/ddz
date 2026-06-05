# 断线重连设计

## 1. 目的

断线重连用于处理：

- 浏览器刷新。
- 临时网络断开。
- WebSocket 断开。
- 移动端休眠/唤醒。

服务端始终权威。重连后必须从服务端状态恢复。

## 2. 需求

MVP 需求：

- 活跃游戏中保留玩家座位。
- 断线后标记玩家离线。
- 在 `RECONNECT_TTL` 内允许重连。
- 重连后发送玩家专属房间快照。
- 私有手牌只恢复给座位所属玩家。
- 离线期间游戏可通过超时自动动作继续。

默认 TTL：

```text
RECONNECT_TTL=5m
```

## 3. 连接身份

连接由以下信息标识：

- 已认证 user ID。
- room ID。
- connection ID。

建议一个用户在一个房间只保留一个活跃连接。

若出现第二个连接：

- 优先使用最新连接。
- 关闭旧连接，原因 `replaced_by_new_connection`。

## 4. 状态模型

玩家连接状态：

- `online`
- `offline`
- `reconnecting`

房间座位状态：

- 活跃游戏中保持 `playing`。
- 游戏前保持 `ready` 或 `joined`。

断线时不要从活跃游戏移除玩家。

## 5. Redis 重连数据

推荐 key：

```text
reconnect:{user_id}:{room_id}
```

value：

```json
{
  "user_id": "u_abc",
  "room_id": "r_abc",
  "seat_index": 0,
  "game_id": "g_abc",
  "disconnected_at": "2026-06-03T09:00:00Z",
  "expires_at": "2026-06-03T09:05:00Z"
}
```

TTL：

- 等于重连 TTL。

MVP 内存替代：

- 在 room manager map 中保存重连元数据。
- 进程重启后不可恢复。

## 6. 断线流程

1. WebSocket 读写循环发现 close 或心跳失败。
2. Gateway 向房间 Actor 发送 `Disconnect(user_id)`。
3. 房间 Actor 标记玩家离线。
4. 房间 Actor 产生 `player_disconnected`。
5. 保存重连元数据。
6. 如果正轮到该玩家，超时系统照常运行。

广播：

```json
{
  "type": "room.player_disconnected",
  "request_id": null,
  "seq": 30,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "reconnect_ttl_seconds": 300
  }
}
```

## 7. 重连流程

1. 客户端使用相同 room ID 和有效 token 打开 WebSocket。
2. Gateway 校验 token。
3. Gateway 校验用户在该房间有座位。
4. Gateway 替换旧连接。
5. Gateway 向房间 Actor 发送 `Reconnect(user_id)`。
6. 房间 Actor 标记玩家在线。
7. 房间 Actor 产生 `player_reconnected`。
8. 服务端发送玩家专属 `room.snapshot`。

私有快照必须包含：

- 自己当前手牌。
- 当前阶段。
- 当前回合。
- 截止时间。
- 上一手出牌。
- 已揭示底牌。
- 若游戏已结束，则包含结算。

## 8. 客户端重连行为

客户端状态：

- `connected`
- `reconnecting`
- `resyncing`
- `failed`

算法：

1. WebSocket close 后显示重连提示。
2. 指数退避重连：
   - 1s。
   - 2s。
   - 4s。
   - 8s。
   - 最大 10s。
3. socket 打开后等待 `room.snapshot`。
4. 使用快照替换本地房间状态。
5. 清空已选牌。

除非明确支持幂等重放，否则重连后不要发送旧的本地排队动作。

## 9. 离线期间超时

策略：

- 离线玩家不立即移除。
- 轮到离线玩家超时时：
  - 叫分：自动不叫。
  - 出牌：能不出则自动不出，否则机器人出最小合法牌。

玩家在超时动作后重连：

- 看到更新后的状态。
- 不能撤销超时动作。

## 10. 离线期间游戏结束

如果玩家离线时游戏结束：

- 结算持久化。
- TTL 内重连返回最终结算快照。
- TTL 后用户可通过 HTTP 记录 API 查询。

## 11. 服务端重启

MVP：

- 如果只用内存，活跃游戏可能丢失。

生产：

- Redis 保存房间快照。
- 游戏事件增量持久化。
- 重启后恢复活跃房间，或按策略标记对局异常中止。

## 12. 安全

重连必须校验：

- JWT 有效。
- user ID 匹配座位归属。
- 房间存在。
- 用户有权接收私有手牌。

不得只凭 room ID 和 seat index 重连。

## 13. 验收标准

- 游戏中刷新浏览器可恢复状态。
- 重连玩家看到自己的当前手牌。
- 其他玩家看到在线状态变化。
- 离线超时动作可执行。
- 隐藏牌不泄漏。

