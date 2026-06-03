# WebSocket 协议规范

## 1. 用途

WebSocket 是房间生命周期和实时游戏动作的传输协议。

HTTP 用于进入房间并获取 WebSocket URL。连接建立后，准备、叫分、出牌、不出等动作都通过 WebSocket 发送。

## 2. 连接地址

```text
GET /ws/v1/rooms/{room_id}?token=<access_token>
```

生产优先方案：

- 支持时使用 WebSocket Upgrade 的 `Authorization` header。
- 浏览器 MVP 可使用 query token，但反向代理和应用日志不得记录该 token。

## 3. 协议信封

所有消息使用 JSON。

客户端到服务端：

```json
{
  "type": "room.ready",
  "request_id": "req_123",
  "seq": 1,
  "payload": {}
}
```

字段：

- `type`：消息类型。
- `request_id`：客户端生成的请求 ID。
- `seq`：客户端递增序号，用于诊断。
- `payload`：类型相关负载。

服务端到客户端：

```json
{
  "type": "ack",
  "request_id": "req_123",
  "seq": 10,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "ok": true
  }
}
```

字段：

- `type`：消息类型。
- `request_id`：响应客户端请求时复制该值。
- `seq`：服务端递增序号。
- `server_time`：ISO 时间。
- `payload`：类型相关负载。

## 4. 连接生命周期

### 4.1 建连

服务端必须：

1. 校验 token。
2. 校验用户属于该房间；若没有旁观模式，则禁止旁观。
3. 绑定连接、用户和房间。
4. 向当前用户发送玩家专属房间快照。
5. 向其他玩家广播在线状态变化。

首条服务端消息：

```json
{
  "type": "room.snapshot",
  "request_id": null,
  "seq": 1,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "room": {},
    "game": {},
    "me": {}
  }
}
```

### 4.2 心跳

客户端发送：

```json
{
  "type": "ping",
  "request_id": "req_ping_1",
  "seq": 2,
  "payload": {
    "client_time": "2026-06-03T09:00:01Z"
  }
}
```

服务端回复：

```json
{
  "type": "pong",
  "request_id": "req_ping_1",
  "seq": 2,
  "server_time": "2026-06-03T09:00:01Z",
  "payload": {}
}
```

心跳规则：

- 客户端每 15 秒发送一次。
- 服务端 45 秒内没有读到消息/心跳则认为连接失效。

### 4.3 断开

服务端必须：

- 标记玩家连接离线。
- 活跃游戏中保留座位。
- 广播玩家离线事件。
- 开始重连 TTL。

## 5. 房间消息

### 5.1 room.ready

客户端：

```json
{
  "type": "room.ready",
  "request_id": "req_1",
  "seq": 3,
  "payload": {
    "ready": true
  }
}
```

服务端事件：

```json
{
  "type": "room.player_ready",
  "request_id": "req_1",
  "seq": 4,
  "server_time": "2026-06-03T09:00:02Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "ready": true
  }
}
```

### 5.2 room.chat_emote

MVP 可选。

```json
{
  "type": "room.chat_emote",
  "request_id": "req_2",
  "seq": 4,
  "payload": {
    "emote": "hello"
  }
}
```

## 6. 游戏消息

### 6.1 game.bid

客户端：

```json
{
  "type": "game.bid",
  "request_id": "req_bid_1",
  "seq": 5,
  "payload": {
    "score": 1
  }
}
```

允许分数：

- `0`：不叫。
- `1`
- `2`
- `3`

服务端事件：

```json
{
  "type": "game.bid_placed",
  "request_id": "req_bid_1",
  "seq": 11,
  "server_time": "2026-06-03T09:00:05Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "score": 1,
    "next_seat_index": 1,
    "deadline_at": "2026-06-03T09:00:20Z"
  }
}
```

### 6.2 game.play_cards

客户端：

```json
{
  "type": "game.play_cards",
  "request_id": "req_play_1",
  "seq": 8,
  "payload": {
    "cards": ["S3", "H3", "D3"]
  }
}
```

广播给所有玩家：

```json
{
  "type": "game.cards_played",
  "request_id": "req_play_1",
  "seq": 20,
  "server_time": "2026-06-03T09:01:00Z",
  "payload": {
    "user_id": "u_abc",
    "seat_index": 0,
    "cards": ["S3", "H3", "D3"],
    "card_group": {
      "type": "triple",
      "rank": 3,
      "length": 3
    },
    "remaining_count": 14,
    "next_seat_index": 1,
    "deadline_at": "2026-06-03T09:01:15Z"
  }
}
```

只发送给出牌者：

```json
{
  "type": "game.my_hand_updated",
  "request_id": "req_play_1",
  "seq": 21,
  "server_time": "2026-06-03T09:01:00Z",
  "payload": {
    "cards": ["S4", "H4"]
  }
}
```

### 6.3 game.pass

客户端：

```json
{
  "type": "game.pass",
  "request_id": "req_pass_1",
  "seq": 9,
  "payload": {}
}
```

服务端事件：

```json
{
  "type": "game.player_passed",
  "request_id": "req_pass_1",
  "seq": 22,
  "server_time": "2026-06-03T09:01:10Z",
  "payload": {
    "user_id": "u_def",
    "seat_index": 1,
    "next_seat_index": 2,
    "deadline_at": "2026-06-03T09:01:25Z"
  }
}
```

### 6.4 game.hint

MVP 可由前端本地提示，也可服务端生成。

服务端方案：

```json
{
  "type": "game.hint",
  "request_id": "req_hint_1",
  "seq": 10,
  "payload": {}
}
```

响应：

```json
{
  "type": "game.hint_result",
  "request_id": "req_hint_1",
  "seq": 23,
  "server_time": "2026-06-03T09:01:12Z",
  "payload": {
    "suggestions": [
      ["S5"],
      ["S6", "H6"]
    ]
  }
}
```

## 7. 服务端广播

### 7.1 room.snapshot

快照是玩家专属的。

```json
{
  "type": "room.snapshot",
  "request_id": null,
  "seq": 30,
  "server_time": "2026-06-03T09:01:20Z",
  "payload": {
    "room": {
      "room_id": "r_abc",
      "mode": "classic",
      "status": "playing",
      "base_score": 1
    },
    "players": [
      {
        "user_id": "u_1",
        "display_name": "A",
        "seat_index": 0,
        "role": "landlord",
        "status": "online",
        "ready": true,
        "remaining_count": 17,
        "is_robot": false
      }
    ],
    "game": {
      "game_id": "g_abc",
      "phase": "playing",
      "current_seat_index": 0,
      "landlord_seat_index": 0,
      "bottom_cards": ["BJ", "S2", "H8"],
      "last_play": null,
      "multiplier": 3,
      "deadline_at": "2026-06-03T09:01:35Z"
    },
    "me": {
      "user_id": "u_1",
      "seat_index": 0,
      "hand": ["S3", "H3"]
    }
  }
}
```

安全要求：

- 只有 `me.hand` 包含私有手牌。
- 其他玩家只暴露 `remaining_count`。

### 7.2 game.started

```json
{
  "type": "game.started",
  "request_id": null,
  "seq": 5,
  "server_time": "2026-06-03T09:00:03Z",
  "payload": {
    "game_id": "g_abc",
    "phase": "bidding",
    "current_seat_index": 0,
    "deadline_at": "2026-06-03T09:00:18Z",
    "my_hand": ["S3", "H3"]
  }
}
```

此消息必须私发，或使用玩家专属 payload。

### 7.3 game.landlord_decided

```json
{
  "type": "game.landlord_decided",
  "request_id": null,
  "seq": 15,
  "server_time": "2026-06-03T09:00:40Z",
  "payload": {
    "landlord_seat_index": 1,
    "landlord_user_id": "u_def",
    "bottom_cards": ["BJ", "S2", "H8"],
    "multiplier": 3,
    "current_seat_index": 1,
    "deadline_at": "2026-06-03T09:00:55Z"
  }
}
```

地主还会收到 `game.my_hand_updated`，其中包含加入底牌后的完整手牌。

### 7.4 game.ended

```json
{
  "type": "game.ended",
  "request_id": null,
  "seq": 80,
  "server_time": "2026-06-03T09:08:00Z",
  "payload": {
    "winner_side": "landlord",
    "winner_user_id": "u_def",
    "settlements": [
      {
        "user_id": "u_def",
        "seat_index": 1,
        "role": "landlord",
        "score_delta": 24
      }
    ],
    "final_multiplier": 12,
    "reason": {
      "base_score": 1,
      "bid_score": 3,
      "bomb_count": 1,
      "rocket_count": 0
    }
  }
}
```

## 8. Ack 与错误

Ack：

```json
{
  "type": "ack",
  "request_id": "req_1",
  "seq": 100,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "ok": true
  }
}
```

如果马上会发送同 request_id 的事件，ack 可以省略。

错误：

```json
{
  "type": "error",
  "request_id": "req_play_1",
  "seq": 101,
  "server_time": "2026-06-03T09:00:00Z",
  "payload": {
    "code": "invalid_card_set",
    "message": "invalid card set"
  }
}
```

## 9. 顺序与幂等

- 服务端序号按连接递增。
- 房间事件序号按对局递增。
- 客户端 request_id 用于关联和短期幂等。
- 短时间内重复 request_id 应返回前次结果或拒绝为重复请求。

## 10. 超时处理

服务端事件：

```json
{
  "type": "game.timeout",
  "request_id": null,
  "seq": 40,
  "server_time": "2026-06-03T09:02:00Z",
  "payload": {
    "seat_index": 2,
    "action": "auto_pass"
  }
}
```

规则：

- 叫分超时默认不叫或最低合法动作。
- 出牌超时若可不出则自动不出。
- 不可不出时，机器人策略选择最小合法出牌。

## 11. 验收标准

- 客户端重连后收到 `room.snapshot`。
- 非当前回合动作返回 `error`。
- 其他玩家永远收不到 `me.hand`。
- 所有状态变更消息都由服务端房间 Actor 校验。
- 超时事件对所有玩家可见。

