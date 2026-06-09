# Load Smoke Test

`T-10003` 用于验证当前开发环境下的基础承载能力，重点覆盖：
- 批量游客登录
- 批量创建房间 / 加入房间
- 批量建立 WebSocket 连接
- 可选少量房间动作冒烟（`room.ready`）

## 1. 启动后端

```bash
cd backend
go run ./cmd/server
```

默认地址：`http://127.0.0.1:8080`

## 2. 执行负载冒烟

```bash
cd backend
go run ./cmd/loadtest -base-url http://127.0.0.1:8080 -connections 100 -concurrency 20 -hold 5s
```

参数说明：

- `-base-url`：后端 HTTP 地址
- `-connections`：目标 WebSocket 连接总数
- `-concurrency`：并发房间工作协程数
- `-hold`：连接建立后保持时长
- `-ready-rooms`：额外发送 `room.ready` 动作的房间数
- `-mode`：房间模式，默认 `classic`
- `-base-score`：底分，默认 `1`

## 3. 1000 Idle WebSocket 目标示例

```bash
cd backend
go run ./cmd/loadtest -base-url http://127.0.0.1:8080 -connections 1000 -concurrency 80 -hold 15s
```

说明：
- `connections=1000` 会按 3 人房自动拆分为多个房间。
- 这是开发环境的冒烟基线，不是正式容量结论。
- 若本机句柄数、端口资源、杀毒软件或 Docker 网络有限，可能先被环境限制。

## 4. 成功判定

命令退出码为 `0`，并输出 JSON 汇总，关键字段：

- `target_connections`
- `connected_connections`
- `created_rooms`
- `joined_players`
- `ready_actions`
- `failure_count`
- `failures`

当 `connected_connections == target_connections` 且 `failure_count == 0` 时，视为本轮通过。
