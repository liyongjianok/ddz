# 生产架构

## 1. 生产目标

从 MVP 单进程部署演进到生产级棋牌游戏架构：

- 稳定 WebSocket 连接。
- 服务端权威房间状态。
- API 可水平扩展。
- 游戏事件可观测。
- 房间状态可恢复。
- 玩法、持久化和运维边界清晰。

## 2. MVP 与生产

### 2.1 MVP

```text
Browser -> Single Go Server -> PostgreSQL
                         -> Redis optional
```

特征：

- HTTP 和 WebSocket 在同一进程。
- 活跃房间在内存。
- 对局完成后持久化记录。
- 机器人服务在同一进程。

### 2.2 生产

```text
Browser
  -> CDN / Reverse Proxy
  -> API Service
  -> WebSocket Gateway
  -> Room Service / Room Actors
  -> PostgreSQL / Redis / Message Broker
```

特征：

- HTTP 和 WebSocket 可独立扩展。
- 房间归属注册。
- Redis 保存 session、重连和活跃快照。
- 事件持久化和分析管道。

## 3. 生产组件

### 3.1 API Service

职责：

- 认证。
- 大厅。
- 匹配入口。
- 记录。
- 轻量管理 API。

应尽量无状态，仅依赖数据库/Redis。

### 3.2 WebSocket Gateway

职责：

- 连接 upgrade。
- token 校验。
- 心跳。
- 消息 schema 校验。
- 连接映射。
- 将命令转发到房间 Actor。
- 向客户端发送事件。

扩展：

- 按 user ID 或 room ID 粘性路由。
- 跨节点路由需要 Redis pub/sub、NATS 或类似组件。

### 3.3 Room Service

职责：

- 房间 Actor 生命周期。
- 串行命令处理。
- 游戏状态。
- 定时器。
- 机器人命令接入。
- 快照生成。

扩展策略：

- 每个活跃房间同一时刻只归属一个节点。
- Redis 保存 `room_owner:{room_id} -> node_id`。
- 远程房间命令路由到 owner node。

### 3.4 Rules Engine

职责：

- 牌解析。
- 牌型识别。
- 出牌比较。
- 合法出牌生成。
- 结算。

必须保持纯库代码。

### 3.5 Robot Service

MVP：

- 进程内服务。

生产：

- 可继续进程内，也可拆为 worker pool。
- 必须通过房间命令协议提交动作。

### 3.6 Persistence Service

职责：

- 写入游戏事件。
- 写入结算。
- 更新玩家统计。
- 查询对局记录。

结算使用事务。

## 4. 数据存储

### 4.1 PostgreSQL

保存：

- 用户。
- 资料。
- 房间元数据。
- 对局。
- 对局事件。
- 结算。
- 审计日志。

### 4.2 Redis

保存：

- 会话缓存。
- WebSocket 连接元数据。
- 用户活跃房间映射。
- 房间归属映射。
- 活跃房间快照。
- 重连元数据。
- 限流计数器。

### 4.3 消息队列

MVP 可不使用，生产推荐：

- NATS。
- Redis Streams。
- Kafka 适合后续重分析场景。

用途：

- 跨节点房间命令。
- 事件 fanout。
- 分析数据采集。

## 5. 扩展模型

### 5.1 房间归属

一个房间同一时刻只由一个节点拥有。

归属 key：

```text
room_owner:{room_id} = node_id
```

要求：

- owner 周期性续租。
- owner 宕机时，恢复流程使用最新快照/事件。

### 5.2 粘性会话

方案：

1. 将房间内 WebSocket 连接全部路由到 owner node。
2. Gateway node 接收连接，再将命令/事件转发到 room owner。

MVP 使用单节点，相当于方案 1。

生产应为方案 2 预留。

## 6. 可靠性

### 6.1 优雅停机

停机时：

1. 标记节点 draining。
2. 停止接受新房间。
3. 尽可能通知客户端。
4. 持久化房间快照。
5. 完成活跃命令处理。
6. 释放房间归属租约。

### 6.2 房间恢复

恢复输入：

- Redis 最新房间快照。
- 已持久化游戏事件。
- 重连元数据。

恢复策略：

- 快照足够新则恢复。
- 状态无法安全恢复时，中止并标记 no-contest 或异常对局。

### 6.3 事件耐久

建议：

- 游戏过程中持续持久化关键事件。
- 至少持久化 game start 和 game end。
- 生产应在动作发生时追加事件。

## 7. 安全和反作弊

### 7.1 服务端权威

服务端拥有：

- 洗牌。
- 发牌。
- 手牌。
- 回合。
- 动作校验。
- 结算。

### 7.2 隐藏信息

保护：

- 玩家专属快照。
- DTO 严格分离。
- 不广播完整状态。
- 日志脱敏。

### 7.3 限流

限流范围：

- 登录。
- HTTP API。
- 每连接 WebSocket 消息。
- 非法游戏动作。

### 7.4 审计

可疑行为：

- 大量非法动作。
- 坏牌频繁断线。
- 同 IP 多账号。
- 异常胜率。

MVP 只记录日志；未来可增加评分模型。

## 8. 可观测性

### 8.1 Metrics

推荐指标：

- 在线用户。
- 活跃房间。
- WebSocket 连接数。
- 每秒消息数。
- 每分钟非法动作数。
- 房间 Actor 队列长度。
- 命令处理延迟。
- 重连成功率。
- 对局完成数。
- 平均对局时长。

### 8.2 日志

结构化日志：

- trace ID。
- user ID。
- room ID。
- game ID。
- event type。
- severity。

### 8.3 链路追踪

追踪：

- HTTP 请求。
- WebSocket 命令处理。
- Room Actor 命令。
- DB 写入。

## 9. 性能目标

MVP：

- 开发基准单节点 1,000 idle WebSocket 连接。
- 单节点 100 活跃房间。
- 规则校验低于 5 ms。

生产初期：

- 每个 gateway 节点 10,000 idle WebSocket 连接，取决于实例规格。
- 每个 room-service 节点 1,000 活跃房间，取决于定时器和机器人负载。
- P95 命令处理低于 100 ms。

实际目标必须通过压测验证。

## 10. 发布策略

### 10.1 环境

- local。
- dev。
- staging。
- production。

### 10.2 部署流程

1. 运行单元测试。
2. 运行集成测试。
3. 构建 Docker 镜像。
4. 执行 migrations。
5. 部署 staging。
6. 运行冒烟测试。
7. 部署 production。
8. 观察 metrics 和 logs。

## 11. 生产验收标准

- WebSocket 可通过反向代理工作。
- API 水平扩展不破坏认证/大厅。
- 房间状态不会跨节点并发修改。
- TTL 内重连可用。
- 对局记录持久可靠。
- 日志、API、广播均不泄漏隐藏牌。
- 优雅停机不破坏已完成对局。

