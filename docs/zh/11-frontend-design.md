# 前端设计

## 1. 设计目标

创建一个清晰、快速、适合浏览器游玩的斗地主界面。视觉上应像一张成熟的棋牌游戏桌，但交互优先级高于装饰。

重点：

- 牌面可读。
- 当前回合清晰。
- 操作按钮高效。
- 重连状态明确。
- 不出现隐藏信息泄漏。

## 2. 页面

### 2.1 登录页

目的：

- 让用户快速以游客身份进入。

元素：

- 昵称输入框。
- 游客登录按钮。
- 简洁产品标题。

验收：

- 不注册也能登录。

### 2.2 大厅页

目的：

- 展示玩家资料和房间入口。

元素：

- 顶部栏：昵称、虚拟金币、连接状态。
- 快速开始主按钮。
- 房间列表。
- 模式/底分选择。
- 在线摘要。

验收：

- 快速开始可进入房间。
- 房间列表可扫描。

### 2.3 房间页

目的：

- 主游戏界面。

元素：

- 三个玩家座位。
- 中央出牌区域。
- 底牌区域。
- 当前 trick 展示。
- 自己手牌。
- 操作按钮。
- 倒计时。
- 连接/重连状态。
- 结算弹窗。

验收：

- 用户可以完整打一局。

## 3. 房间布局

桌面布局：

```text
------------------------------------------------
| 顶部栏：房间号、阶段、倍数、倒计时            |
------------------------------------------------
|              上方玩家座位 / 出牌              |
|                                                |
| 左侧玩家          中央牌桌          右侧玩家   |
|                                                |
|              自己出牌 / 操作提示              |
------------------------------------------------
| 自己手牌                                           |
| 操作按钮：准备 / 叫分 / 出牌 / 不出             |
------------------------------------------------
```

移动端：

- 自己手牌固定在底部。
- 对手座位压缩显示在顶部。
- 操作区紧凑。
- 除手牌外尽量避免横向滚动。

## 4. 视觉风格

推荐：

- 桌面背景：深绿色或低饱和青绿色。
- 牌面：白色或暖白。
- 红色花色用红色。
- 黑色花色用近黑色。
- 操作按钮高对比。
- 当前回合使用明显描边或光效。
- 禁用控件要明显变灰。

避免：

- 营销落地页式布局。
- 牌面文字过小。
- 只能 hover 才能发现的按钮。
- 过重动画影响出牌效率。

## 5. 组件

### 5.1 Card

```ts
type CardProps = {
  code: string;
  selected?: boolean;
  disabled?: boolean;
  faceDown?: boolean;
  onClick?: () => void;
};
```

行为：

- 可用时点击切换选中。
- 选中牌向上浮起。
- 背面牌隐藏点数。

### 5.2 Hand

```ts
type HandProps = {
  cards: CardCode[];
  selected: CardCode[];
  sortMode: "rank" | "suit";
  onToggle: (card: CardCode) => void;
};
```

行为：

- 手牌可轻微重叠以适配屏幕。
- 选中牌顺序保持手牌显示顺序。

### 5.3 PlayerSeat

```ts
type PlayerSeatProps = {
  player: PublicPlayer;
  isCurrentTurn: boolean;
  lastPlayedCards: CardCode[];
  countdown?: number;
};
```

展示：

- 头像。
- 昵称。
- 角色。
- 剩余牌数。
- 在线/离线。
- 机器人标识。

### 5.4 ActionPanel

```ts
type ActionPanelProps = {
  phase: GamePhase;
  isMyTurn: boolean;
  canPass: boolean;
  selectedCards: CardCode[];
  onReady: () => void;
  onBid: (score: 0 | 1 | 2 | 3) => void;
  onPlay: () => void;
  onPass: () => void;
  onHint: () => void;
};
```

规则：

- 只展示当前阶段相关操作。
- 非自己回合时禁用出牌类操作。
- 未选牌时禁用出牌按钮。

### 5.5 SettlementModal

展示：

- 胜利方。
- 分数变化。
- 倍数详情。
- 继续/准备按钮。

## 6. 前端状态模型

### 6.1 Auth Store

字段：

- user。
- token。
- status。

动作：

- guestLogin。
- loadMe。
- logout。

### 6.2 Lobby Store

字段：

- summary。
- rooms。
- loading。

动作：

- fetchSummary。
- fetchRooms。
- quickStart。

### 6.3 Room Store

字段：

- room。
- players。
- game。
- me。
- selectedCards。
- connectionState。
- lastError。

动作：

- connect。
- disconnect。
- applySnapshot。
- applyEvent。
- toggleCard。
- clearSelection。
- sendReady。
- sendBid。
- sendPlay。
- sendPass。

## 7. WebSocket 客户端行为

状态：

- `idle`
- `connecting`
- `connected`
- `reconnecting`
- `disconnected`
- `failed`

行为：

- 进入房间页时连接。
- 每 15 秒发送心跳。
- close 后指数退避重连。
- 重连后等待权威 `room.snapshot`。
- 不把旧本地状态当作事实。

## 8. 手牌排序

默认排序：

```text
RJ, BJ, 2, A, K, Q, J, T, 9 ... 3
```

可选排序：

- 按点数数量分组，方便选择牌型。

客户端排序只影响显示，不影响服务端逻辑。

## 9. 错误反馈

示例：

- `not_player_turn`：还没轮到你出牌。
- `invalid_card_set`：这组牌型不合法。
- `cannot_pass`：当前不能不出。
- `connection_lost`：连接断开，正在重连。

不要显示后端堆栈。

## 10. 可访问性

最低要求：

- 按钮有文本或 accessible label。
- 卡牌有可读 label。
- 颜色不是唯一的回合提示。
- 键盘选牌可选但推荐。

## 11. 前端验收标准

- 游客登录可用。
- 大厅快速开始可用。
- 房间能渲染服务端快照。
- 玩家可以准备、叫分、出牌、不出。
- 桌面和移动端牌面可读。
- 重连状态可见。
- 前端状态中不保存其他玩家隐藏牌。

