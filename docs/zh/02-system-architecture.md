# 系统架构设计

## 1. 架构目标

系统需要支撑浏览器实时斗地主，并保持边界清晰：

- HTTP API 负责登录、大厅、房间列表、记录和轻量管理查询。
- WebSocket 网关负责实时房间玩法。
- 领域规则引擎独立于传输层和持久化层。
- 房间 Actor/服务串行处理单个房间内的游戏动作。
- 持久化层保存用户、房间、对局记录和事件。
- 机器人服务通过内部命令像真实玩家一样参与游戏。

架构既要适合 MVP 快速落地，也要能自然演进到生产环境。

## 2. 推荐技术栈

### 2.1 后端

- 语言：Go 1.22+
- HTTP 框架：Gin、Echo 或 chi，推荐 chi。
- WebSocket：`nhooyr.io/websocket` 或 `gorilla/websocket`，推荐 `nhooyr.io/websocket`。
- 数据库：生产推荐 PostgreSQL 15+，本地原型可选 SQLite。
- 缓存/会话：Redis 7+。
- ORM/查询：sqlc 或 GORM，生产推荐 sqlc。
- 配置：环境变量加配置文件。
- 日志：zerolog 或 slog，默认推荐标准库 `log/slog`。
- 测试：Go testing，可选 testify。

### 2.2 前端

- React 18+
- TypeScript
- Vite
- Zustand 或 Redux Toolkit，MVP 推荐 Zustand。
- Tailwind CSS 或 CSS Modules。
- 独立 WebSocket 客户端抽象。

### 2.3 部署

- Docker。
- Docker Compose 用于开发和预发。
- Nginx 或 Caddy 反向代理。
- PostgreSQL。
- Redis。

## 3. 高层组件

```text
Browser Client
  | HTTP
  v
API Server
  | auth/session/db
  v
Database

Browser Client
  | WebSocket
  v
WebSocket Gateway
  | room commands/events
  v
Room Runtime / Game Service
  | rules validation
  v
Rules Engine

Room Runtime
  | events/snapshots
  v
Persistence

Robot Service
  | internal commands
  v
Room Runtime
```

## 4. 后端目录结构

推荐 Go 目录：

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   ├── auth/
│   ├── user/
│   ├── lobby/
│   ├── room/
│   ├── game/
│   ├── robot/
│   ├── ws/
│   ├── record/
│   ├── storage/
│   └── observability/
├── pkg/
│   └── errors/
└── tests/
```

建议文件：

```text
internal/app/server.go
internal/app/config.go
internal/auth/jwt.go
internal/auth/service.go
internal/auth/middleware.go
internal/room/actor.go
internal/room/manager.go
internal/room/snapshot.go
internal/game/card.go
internal/game/deck.go
internal/game/rules.go
internal/game/compare.go
internal/game/state.go
internal/game/event.go
internal/game/settlement.go
internal/ws/gateway.go
internal/ws/connection.go
internal/ws/protocol.go
```

依赖规则：

- `internal/game` 不依赖 HTTP、WebSocket、数据库、Redis 或前端 DTO。
- `internal/room` 可依赖 `internal/game`、持久化接口和机器人接口。
- `internal/ws` 只负责协议消息和房间命令之间的转换。
- `internal/robot` 可读取游戏快照，但不能直接修改游戏状态。

## 5. 前端目录结构

```text
frontend/
├── src/
│   ├── app/
│   ├── api/
│   ├── ws/
│   ├── store/
│   ├── domain/
│   ├── pages/
│   ├── components/
│   └── styles/
└── tests/
```

前端规则：

- 客户端可以选择牌和发送动作，但服务端负责最终校验。
- 客户端基于服务端快照和事件渲染。
- 客户端不得推断或保存其他玩家隐藏手牌。
- 重连后必须请求权威快照。

## 6. 房间运行时模型

每个房间按 Actor 模型运行：

- 每个房间一个命令队列。
- 命令串行处理。
- 不允许并发修改同一房间状态。
- 状态变化后产生事件。
- 给不同玩家发送玩家专属快照。

命令示例：

- JoinRoom。
- LeaveRoom。
- Ready。
- Bid。
- PlayCards。
- Pass。
- Timeout。
- Disconnect。
- Reconnect。

事件示例：

- RoomJoined。
- PlayerReady。
- GameStarted。
- BidPlaced。
- LandlordDecided。
- CardsPlayed。
- PlayerPassed。
- TurnChanged。
- GameEnded。

## 7. 状态归属

### 7.1 服务端状态

- 牌堆。
- 洗牌种子或哈希。
- 玩家手牌。
- 当前回合。
- 上一手有效出牌。
- 叫分历史。
- 超时截止时间。
- 结算结果。

### 7.2 客户端状态

- 当前选中的牌。
- UI 排序偏好。
- 本地动画状态。
- 临时提示和视觉状态。

客户端状态不能作为游戏事实。

## 8. 持久化策略

MVP 可将活跃房间状态放在内存中，并持久化已完成对局事件。

生产推荐：

- Redis 保存活跃房间注册、玩家连接映射和短期重连状态。
- 数据库保存持久对局记录。
- 可选事件流用于分析。

事件持久化建议：

- 重要游戏事件以追加形式保存。
- 保存最终快照用于快速查询。
- 游戏完成和结算使用事务。

## 9. API 与 WebSocket 扩展

MVP：

- 单进程同时处理 HTTP 和 WebSocket。

生产：

- 多个 HTTP 实例。
- WebSocket 网关按 room_id 或 player_id 做粘性路由。
- Redis pub/sub 或消息队列处理跨节点事件。
- 房间归属注册表确保命令送达房间所在节点。

## 10. 错误处理

使用稳定错误：

- `ErrUnauthorized`
- `ErrRoomNotFound`
- `ErrSeatUnavailable`
- `ErrGameAlreadyStarted`
- `ErrNotPlayerTurn`
- `ErrInvalidBid`
- `ErrInvalidCardSet`
- `ErrCannotPass`
- `ErrStateConflict`

协议响应包含：

- code。
- message。
- request_id。

不向客户端暴露内部堆栈。

## 11. 配置项

```text
APP_ENV
HTTP_ADDR
DATABASE_URL
REDIS_URL
JWT_SECRET
ACCESS_TOKEN_TTL
ROOM_IDLE_TTL
RECONNECT_TTL
TURN_TIMEOUT_SECONDS
ROBOT_FILL_DELAY_SECONDS
LOG_LEVEL
```

## 12. 研发里程碑

1. 规则引擎和测试。
2. 内存房间 Actor。
3. HTTP 认证和大厅 API。
4. WebSocket 网关和协议。
5. 前端登录/大厅/房间。
6. 机器人接入。
7. 断线重连。
8. 持久化和记录。
9. 部署和可观测性。

## 13. 架构验收标准

- 规则引擎可以脱离服务端单独单元测试。
- 房间命令串行处理。
- WebSocket 消息 schema 稳定。
- 隐藏牌绝不广播给其他玩家。
- 已完成对局可由持久化事件重放。
- 本地开发可一条命令启动后端。

