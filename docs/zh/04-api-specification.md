# HTTP API 规范

## 1. API 原则

- HTTP API 负责认证、大厅、房间元数据、记录和轻量管理查询。
- 实时游戏动作使用 WebSocket，不使用 HTTP。
- 所有受保护接口都需要 Bearer JWT。
- 响应格式保持一致。
- HTTP 接口不得暴露隐藏牌。

## 2. Base URL

```text
/api/v1
```

## 3. 通用 Header

```text
Authorization: Bearer <access_token>
Content-Type: application/json
X-Request-ID: <uuid optional>
```

## 4. 通用响应

成功：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {},
  "request_id": "req_123"
}
```

错误：

```json
{
  "code": "room_not_found",
  "message": "room not found",
  "data": null,
  "request_id": "req_123"
}
```

## 5. 错误码

```text
ok
bad_request
unauthorized
forbidden
not_found
rate_limited
internal_error
room_not_found
seat_unavailable
game_already_started
already_in_room
not_in_room
```

## 6. 认证 API

### 6.1 游客登录

```text
POST /api/v1/auth/guest
```

请求：

```json
{
  "display_name": "Guest123",
  "avatar_url": ""
}
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "user": {
      "id": "u_abc",
      "display_name": "Guest123",
      "avatar_url": "",
      "account_type": "guest"
    },
    "access_token": "jwt",
    "expires_in": 86400
  },
  "request_id": "req_123"
}
```

规则：

- 昵称为空时服务端自动生成。
- 游客用户也需要持久化，确保对局记录稳定。

### 6.2 当前用户

```text
GET /api/v1/auth/me
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "id": "u_abc",
    "display_name": "Guest123",
    "avatar_url": "",
    "account_type": "guest",
    "profile": {
      "level": 1,
      "coin_balance": 10000,
      "total_games": 0,
      "wins": 0
    }
  },
  "request_id": "req_123"
}
```

## 7. 大厅 API

### 7.1 大厅摘要

```text
GET /api/v1/lobby/summary
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "online_players": 123,
    "active_rooms": 45,
    "modes": [
      {
        "mode": "classic",
        "base_score": 1,
        "online_players": 100,
        "waiting_rooms": 8
      }
    ]
  },
  "request_id": "req_123"
}
```

### 7.2 房间列表

```text
GET /api/v1/rooms?mode=classic&status=waiting&page=1&page_size=20
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "items": [
      {
        "room_id": "r_abc",
        "mode": "classic",
        "status": "waiting",
        "base_score": 1,
        "player_count": 2,
        "max_players": 3,
        "created_at": "2026-06-03T09:00:00Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "request_id": "req_123"
}
```

### 7.3 快速开始

```text
POST /api/v1/matchmaking/quick-start
```

请求：

```json
{
  "mode": "classic",
  "base_score": 1
}
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 0,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

规则：

- 用户已有活跃房间时返回该房间。
- 匹配逻辑不得让一个用户同时进入两个房间。

### 7.4 创建房间

```text
POST /api/v1/rooms
```

请求：

```json
{
  "mode": "classic",
  "base_score": 1,
  "private": false
}
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 0,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

### 7.5 加入房间

```text
POST /api/v1/rooms/{room_id}/join
```

请求：

```json
{
  "preferred_seat": null
}
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "seat_index": 2,
    "ws_url": "/ws/v1/rooms/r_abc"
  },
  "request_id": "req_123"
}
```

规则：

- 不能加入已关闭房间。
- 满员房间拒绝加入，除非允许替换游戏开始前的未锁定机器人。

### 7.6 离开房间

```text
POST /api/v1/rooms/{room_id}/leave
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "room_id": "r_abc",
    "left": true
  },
  "request_id": "req_123"
}
```

规则：

- 游戏开始前允许离开。
- 游戏进行中 HTTP leave 不应直接改变胜负；MVP 可以只标记断线，不实现投降。

## 8. 记录 API

### 8.1 我的对局记录

```text
GET /api/v1/records/my?page=1&page_size=20
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "items": [
      {
        "game_id": "g_abc",
        "mode": "classic",
        "role": "landlord",
        "winner_side": "landlord",
        "score_delta": 24,
        "started_at": "2026-06-03T09:00:00Z",
        "ended_at": "2026-06-03T09:08:00Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "request_id": "req_123"
}
```

### 8.2 对局详情

```text
GET /api/v1/records/{game_id}
```

响应：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "game_id": "g_abc",
    "room_id": "r_abc",
    "mode": "classic",
    "base_score": 1,
    "multiplier": 12,
    "winner_side": "landlord",
    "players": [
      {
        "user_id": "u_1",
        "display_name": "A",
        "seat_index": 0,
        "role": "landlord",
        "score_delta": 24
      }
    ],
    "events": []
  },
  "request_id": "req_123"
}
```

安全：

- MVP 仅允许参与者查看详细记录。
- 完整暗牌回放只允许对局结束后向参与者或管理员开放。

## 9. 轻量管理 API

可由管理员角色保护，MVP UI 可不开放。

```text
GET /api/v1/admin/rooms/active
GET /api/v1/admin/games/{game_id}/audit
```

## 10. HTTP 验收标准

- 游客登录返回有效 token。
- 受保护接口拒绝缺失或非法 token。
- 快速开始返回房间和座位。
- 房间列表不包含隐藏牌。
- 对局详情只返回授权数据。
- 错误遵循统一响应格式。

