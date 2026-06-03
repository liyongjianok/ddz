# 部署指南

## 1. 部署目标

提供可靠的本地和生产部署路径：

- 本地开发命令简单。
- 使用 Docker Compose 做开发/预发环境。
- 生产支持后端、前端、PostgreSQL、Redis 和反向代理独立扩展。

## 2. 环境变量

后端：

```text
APP_ENV=dev
HTTP_ADDR=:8080
DATABASE_URL=postgres://ddz:ddz@postgres:5432/ddz?sslmode=disable
REDIS_URL=redis://redis:6379/0
JWT_SECRET=change-me
ACCESS_TOKEN_TTL=24h
ROOM_IDLE_TTL=30m
RECONNECT_TTL=5m
TURN_TIMEOUT_SECONDS=15
ROBOT_FILL_DELAY_SECONDS=5
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=http://localhost:5173
```

前端：

```text
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_BASE_URL=ws://localhost:8080/ws/v1
```

## 3. 本地开发

推荐目录：

```text
backend/
frontend/
deploy/
docs/
```

后端：

```bash
cd backend
go mod tidy
go run ./cmd/server
```

前端：

```bash
cd frontend
npm install
npm run dev
```

数据库：

```bash
docker compose -f deploy/docker-compose.dev.yml up postgres redis
```

## 4. Docker Compose 开发示例

`deploy/docker-compose.dev.yml`：

```yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ddz
      POSTGRES_PASSWORD: ddz
      POSTGRES_DB: ddz
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

完整 stack compose 可在后端和前端实现后补充。

## 5. 后端 Dockerfile

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -o /bin/ddz-server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /bin/ddz-server /bin/ddz-server
EXPOSE 8080
CMD ["/bin/ddz-server"]
```

## 6. 前端 Dockerfile

```dockerfile
FROM node:20-alpine AS build
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY deploy/nginx.frontend.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

## 7. 反向代理

Nginx 必须支持 WebSocket upgrade。

```nginx
server {
    listen 80;
    server_name _;

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /ws/ {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 60s;
    }
}
```

## 8. 数据库迁移

推荐：

```bash
migrate -path backend/internal/storage/migrations -database "$DATABASE_URL" up
```

策略：

- 部署新版后端前先执行 migrations。
- 破坏性变更前备份生产库。
- 不编辑已应用 migration。

## 9. 健康检查

```text
GET /healthz
GET /readyz
```

`/healthz`：

- 进程存活。

`/readyz`：

- 数据库可达。
- Redis 可达。
- 服务可接收流量。

## 10. 日志

生产日志必须是结构化 JSON。

必需字段：

- `time`
- `level`
- `message`
- `event_type`
- `trace_id`
- `user_id`
- `room_id`
- `game_id`

不得记录 token 或普通应用日志中的完整隐藏牌。

## 11. 优雅停机

后端应：

1. 停止接受新 HTTP/WebSocket 连接。
2. 尽量通知活跃 WebSocket 客户端。
3. 完成正在处理的房间命令。
4. 持久化关键游戏状态或标记房间可恢复。
5. 关闭数据库和 Redis 连接。

默认优雅停机超时：

- 30 秒。

## 12. 生产检查清单

- 设置强 `JWT_SECRET`。
- 使用 HTTPS。
- 限制 CORS origins。
- 启用限流。
- 启用数据库备份。
- 配置日志保留。
- 配置 Redis 持久化或明确可接受丢失策略。
- 配置监控告警。
- 验证 WebSocket 代理超时。
- 运行负载冒烟测试。

## 13. 部署验收标准

- 本地开发环境可启动。
- 后端 health check 通过。
- 前端可调用 API。
- WebSocket 可通过代理 upgrade。
- 数据库 migration 干净应用。
- 日志结构化。

