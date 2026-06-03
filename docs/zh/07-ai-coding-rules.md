# AI 编码规则

## 1. 目的

本文档是 ChatGPT、Claude Code、Codex 等 AI 编程代理生成本项目代码时的操作手册。

目标是可控的增量开发。不要一次生成整个项目。

## 2. 黄金规则

1. 写代码前先读相关文档。
2. 每次只实现一个边界明确的模块。
3. 游戏规则必须服务端权威。
4. 不得向未授权客户端暴露隐藏牌。
5. 规则和状态流转必须写测试。
6. 偏好简单明确的代码，不写炫技抽象。
7. 遵守架构文档中的包边界。
8. 不更新文档就不得发明 API 或协议字段。
9. 不得加入真金钱语义。
10. 不得静默修改斗地主规则。

## 3. 每个模块的必需流程

每个编码任务：

1. 确认目标模块。
2. 阅读相关文档。
3. 列出输入和输出。
4. 实现代码。
5. 添加测试。
6. 运行测试。
7. 总结变更文件和验证结果。

## 4. 后端编码标准

### 4.1 Go 风格

- 使用 Go 1.22+。
- 使用惯用包名。
- 避免全局可变状态，配置常量除外。
- 通过构造函数做依赖注入。
- 必要时返回类型化错误。
- 函数保持可测试。
- IO 和长耗时操作使用 `context.Context`。
- 持久化时间使用 UTC。

### 4.2 包依赖规则

允许：

- `room` 依赖 `game`。
- `ws` 依赖 `room`。
- API handler 依赖 service。
- `robot` 依赖游戏快照和策略接口。

禁止：

- `game` 依赖 `ws`、`http`、`db`、`redis`。
- `game` 读取环境变量。
- `game` 记录传输层细节日志。
- 后端领域层导入前端 DTO。

### 4.3 错误处理

稳定错误码：

```text
unauthorized
room_not_found
not_player_turn
invalid_bid
invalid_card_set
cannot_pass
state_conflict
internal_error
```

不得向客户端暴露堆栈。

### 4.4 日志

使用结构化日志：

- event_type。
- trace_id。
- user_id。
- room_id。
- game_id。
- request_id。

禁止记录：

- JWT token。
- password hash。
- 普通应用日志中的完整隐藏手牌。

## 5. 前端编码标准

### 5.1 TypeScript

- 开启 strict。
- 协议类型统一定义。
- 避免 `any`。
- API/WebSocket 客户端与 UI 组件分离。

### 5.2 UI 状态

状态分类：

- Auth state。
- Lobby state。
- Room snapshot state。
- Local card selection state。
- Connection state。

不得保存其他玩家隐藏牌。

### 5.3 UX 规则

- 始终显示当前阶段。
- 始终显示当前回合。
- 始终显示倒计时。
- 禁用非法动作按钮。
- 展示服务端错误反馈。
- 重连时在收到快照前显示同步状态。

## 6. 游戏规则编码规则

### 6.1 确定性

规则函数应是确定性的：

```go
Recognize(cards []Card) (CardGroup, error)
CanBeat(candidate CardGroup, previous CardGroup) bool
LegalMoves(hand []Card, previous *CardGroup) []CardGroup
```

### 6.2 测试矩阵

规则实现必须测试：

- 单张。
- 对子。
- 三张。
- 三带一。
- 三带二。
- 顺子。
- 连对。
- 飞机。
- 炸弹。
- 火箭。
- 非法牌型。
- 炸弹压非炸弹。
- 火箭压炸弹。
- 同牌型高点数压低点数。
- 不同非炸弹牌型不能比较。

## 7. WebSocket 编码规则

- Upgrade 时校验 token。
- 一个连接绑定一个用户和一个房间。
- 所有入站消息必须做 schema 校验。
- 未知消息类型返回 error。
- 使用 request_id 关联响应。
- 发送玩家专属快照。
- 不广播私有手牌。
- 处理断线和重连。
- 对动作消息限流。

## 8. 房间 Actor 编码规则

- 每个房间只有一个串行命令路径。
- 不并发修改房间状态。
- 所有房间状态变化都产生事件。
- 定时器事件也进入同一个命令队列。
- 机器人动作也进入同一个命令队列。
- Room manager 负责房间生命周期。

## 9. 数据库编码规则

- 使用 migrations。
- 游戏完成和结算使用事务。
- 事件日志追加写。
- 不更新历史 game_events。
- 完整手牌只保存在对局记录中，不暴露给公开房间 API。

## 10. 测试要求

### 10.1 单元测试

必须覆盖：

- 牌解析。
- 牌堆生成。
- 洗牌/发牌。
- 牌型识别。
- 出牌比较。
- 叫分规则。
- 回合流转。
- 结算。

### 10.2 集成测试

必须覆盖：

- 游客登录。
- 快速开始。
- WebSocket 连接。
- 准备后开始游戏。
- 可行时使用脚本动作完成整局。
- 重连快照。

### 10.3 前端测试

推荐覆盖：

- 卡牌渲染。
- 选牌行为。
- 动作按钮状态。
- WebSocket 事件 reducer。
- 重连 UI 状态。

## 11. AI Agent Prompt 模板

```text
你正在实现模块：<module name>。
请先阅读：
- docs/zh/<doc1>
- docs/zh/<doc2>

约束：
- 遵守 docs/zh/07-ai-coding-rules.md。
- 不实现无关模块。
- 添加测试。
- 不得泄漏隐藏牌。

交付：
- 代码文件。
- 测试。
- 简短验证总结。
```

## 12. 推荐生成顺序

1. Go 项目脚手架。
2. `internal/game` card/deck。
3. `internal/game` 牌型识别。
4. `internal/game` 出牌比较。
5. `internal/game` 状态机。
6. 内存房间 Actor。
7. HTTP 认证和大厅。
8. WebSocket 协议。
9. 前端脚手架。
10. 前端房间 UI。
11. 机器人策略。
12. 断线重连。
13. 持久化。
14. 部署。

## 13. 反模式

避免：

- 一个巨大的 `main.go`。
- WebSocket handler 直接修改手牌。
- 前后端重复实现权威规则。
- 只用含糊展示字符串存储牌。
- 向所有客户端广播完整状态。
- 测试依赖 sleep。
- 测试中的随机行为没有固定 seed。
- 使用 cash、deposit、withdraw 等真金钱命名。

## 14. AI 输出验收标准

AI 生成模块只有满足以下条件才可接受：

- 能编译。
- 有聚焦测试。
- 遵守包边界。
- 不引入隐藏牌泄漏。
- 包含实现和测试摘要。

