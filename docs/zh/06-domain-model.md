# 领域模型

## 1. 领域目标

领域模型用于定义游戏概念，并独立于传输层、数据库和 UI。

核心原则：

`internal/game` 在相同输入下必须是纯净、确定、可测试的。

## 2. 聚合边界

### 2.1 User

用户表示身份和资料，不属于规则引擎。

字段：

- user_id。
- display_name。
- avatar_url。
- account_type。
- status。

### 2.2 Room

房间负责座位、准备状态、活跃游戏、连接和生命周期。

字段：

- room_id。
- mode。
- status。
- base_score。
- seats。
- current_game。
- created_at。
- updated_at。

房间拥有：

- 座位分配。
- 准备状态。
- 当前游戏实例。
- 重连状态。

### 2.3 Game

Game 表示一局斗地主。

字段：

- game_id。
- phase。
- deck。
- bottom_cards。
- players。
- landlord_seat_index。
- current_seat_index。
- last_play。
- pass_count。
- bidding_state。
- multiplier。
- event_seq。
- started_at。
- ended_at。

Game 拥有：

- 牌。
- 回合顺序。
- 叫分阶段。
- 出牌阶段。
- 结算。

## 3. 值对象

### 3.1 Card

牌编码格式：

```text
S3  黑桃 3
H3  红桃 3
C3  梅花 3
D3  方块 3
BJ  小王
RJ  大王
```

花色：

- `S`：黑桃。
- `H`：红桃。
- `C`：梅花。
- `D`：方块。

点数：

- `3 4 5 6 7 8 9 T J Q K A 2 BJ RJ`

斗地主大小顺序：

```text
3 < 4 < 5 < 6 < 7 < 8 < 9 < T < J < Q < K < A < 2 < BJ < RJ
```

花色不参与大小比较。

### 3.2 CardGroup

表示一个合法牌型。

字段：

- type。
- primary_rank。
- cards。
- length。
- attachments。

牌型：

- `single`
- `pair`
- `triple`
- `triple_with_single`
- `triple_with_pair`
- `straight`
- `pair_straight`
- `airplane`
- `airplane_with_singles`
- `airplane_with_pairs`
- `four_with_two_singles`
- `four_with_two_pairs`
- `bomb`
- `rocket`

### 3.3 Play

字段：

- seat_index。
- user_id。
- cards。
- group。
- created_at。

### 3.4 Bid

字段：

- seat_index。
- user_id。
- score。
- created_at。

分数：

- 0 表示不叫。
- 1、2、3 表示叫分。

## 4. 实体

### 4.1 PlayerState

字段：

- user_id。
- seat_index。
- role。
- hand。
- status。
- is_robot。
- bid_score。
- remaining_count。

状态：

- `joined`
- `ready`
- `playing`
- `offline`
- `left`

角色：

- `landlord`
- `farmer`
- 地主未确定前为空。

### 4.2 BiddingState

字段：

- start_seat_index。
- current_seat_index。
- highest_bid。
- highest_bid_seat_index。
- bids。
- rounds。
- deadline_at。

规则：

- 玩家可以叫 0、1、2、3。
- 非 0 叫分必须大于当前最高分。
- 叫 3 立即确定地主。
- 无人叫分时按配置策略处理。

MVP 建议：

- 全部不叫时重新发牌一次。
- 再次全部不叫时随机指定地主，底分按 1 处理，避免死循环。

### 4.3 PlayingState

字段：

- current_seat_index。
- last_play。
- last_play_seat_index。
- pass_count。
- deadline_at。

规则：

- `last_play` 为空时，当前玩家必须出牌。
- 如果其他两家都不出，上一手出牌玩家重新获得主动权，可出任意合法牌型。
- 玩家只有在响应他人的有效出牌时才能不出。

## 5. 命令

命令表示外部系统对领域/应用服务的意图。

```text
JoinRoom(user_id)
LeaveRoom(user_id)
Ready(user_id, ready)
StartGame()
PlaceBid(user_id, score)
PlayCards(user_id, cards)
Pass(user_id)
HandleTimeout(seat_index)
Disconnect(user_id)
Reconnect(user_id)
```

命令处理属于房间/应用服务。纯规则校验属于 game 包。

## 6. 事件

事件是状态变化后的事实。

```text
RoomCreated
PlayerJoined
PlayerLeft
PlayerReadyChanged
GameStarted
CardsDealt
BidPlaced
LandlordDecided
BottomCardsRevealed
CardsPlayed
PlayerPassed
TurnChanged
TimeoutTriggered
PlayerDisconnected
PlayerReconnected
GameEnded
SettlementCompleted
```

事件字段：

- event_id。
- room_id。
- game_id。
- seq。
- actor_user_id。
- payload。
- created_at。

## 7. 状态机

### 7.1 房间状态

```text
waiting -> playing -> settling -> waiting
waiting -> closed
playing -> closed 仅允许管理员强制关闭或严重异常中止
```

### 7.2 游戏状态

```text
created -> dealing -> bidding -> playing -> ended
created -> aborted
dealing -> aborted
bidding -> aborted
playing -> aborted
```

## 8. 不变量

必须始终成立：

- 一个房间最多 3 个座位。
- 一局游戏必须有 3 个玩家。
- 一副牌必须有 54 张唯一牌。
- 活跃游戏中，每张牌只存在于一个位置：
  - 玩家手牌。
  - 地主确定前的底牌。
  - 已出牌事件历史。
- 当前回合必须指向有效座位。
- 只有当前玩家能叫分或出牌。
- 玩家不能出不在自己手里的牌。
- 出牌必须是合法牌型。
- 响应出牌必须能压过上一手，炸弹/火箭按特殊规则处理。
- 隐藏手牌必须私有。

## 9. DTO 映射

领域对象应映射为不同 DTO。

公开玩家 DTO：

```json
{
  "user_id": "u_1",
  "display_name": "A",
  "seat_index": 0,
  "role": "farmer",
  "status": "online",
  "remaining_count": 17,
  "is_robot": false
}
```

私有玩家 DTO：

```json
{
  "user_id": "u_1",
  "seat_index": 0,
  "hand": ["S3", "H3"]
}
```

不要把公开 DTO 和私有 DTO 混在一起，避免隐藏牌泄漏。

## 10. 领域验收标准

- 牌解析器拒绝非法牌。
- 牌堆生成 54 张唯一牌。
- 发牌结果为 17、17、17 和 3 张底牌。
- 牌型识别覆盖所有 MVP 牌型。
- 出牌比较符合斗地主规则。
- 领域测试不依赖数据库或 WebSocket。

