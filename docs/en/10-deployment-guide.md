# Deployment Guide

## 1. Deployment Goals

Provide a reliable local and production deployment path:

- Local development with simple commands.
- Staging with Docker Compose.
- Production with independently scalable backend, frontend, PostgreSQL, Redis, and reverse proxy.

## 2. Environment Variables

Backend:

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

Frontend:

```text
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_BASE_URL=ws://localhost:8080/ws/v1
```

## 3. Local Development

Recommended layout:

```text
backend/
frontend/
deploy/
docs/
```

Backend:

```bash
cd backend
go mod tidy
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Database:

```bash
docker compose -f deploy/docker-compose.dev.yml up postgres redis
```

## 4. Docker Compose Dev Example

Create `deploy/docker-compose.dev.yml`:

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

Full stack compose can be added after backend/frontend are implemented.

## 5. Backend Dockerfile

Recommended:

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

## 6. Frontend Dockerfile

Recommended:

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

## 7. Reverse Proxy

Nginx must support WebSocket upgrade.

Example:

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

## 8. Migrations

Recommended tool:

```bash
migrate -path backend/internal/storage/migrations -database "$DATABASE_URL" up
```

Policy:

- Run migrations before deploying new backend.
- Back up production database before destructive migrations.
- Do not edit already-applied migrations.

## 9. Health Checks

Endpoints:

```text
GET /healthz
GET /readyz
```

`/healthz`:

- Process is alive.

`/readyz`:

- Database reachable.
- Redis reachable if configured.
- Server ready to accept traffic.

## 10. Logging

Production logs must be structured JSON.

Required fields:

- `time`
- `level`
- `message`
- `event_type`
- `trace_id`
- `user_id`
- `room_id`
- `game_id`

Never log tokens or full hidden cards in normal application logs.

## 11. Graceful Shutdown

Backend should:

1. Stop accepting new HTTP/WebSocket connections.
2. Notify active WebSocket clients if possible.
3. Finish in-flight room commands.
4. Persist critical game state or mark rooms recoverable.
5. Close database and Redis connections.

Timeout:

- Default graceful shutdown timeout: 30 seconds.

## 12. Production Checklist

Before production:

- Set strong `JWT_SECRET`.
- Use HTTPS.
- Restrict CORS origins.
- Enable rate limiting.
- Enable database backups.
- Configure log retention.
- Configure Redis persistence or acceptable loss policy.
- Configure monitoring alerts.
- Verify WebSocket proxy timeout.
- Run load smoke test.

## 13. Deployment Acceptance Criteria

- Local dev environment starts.
- Backend health check passes.
- Frontend can call API.
- WebSocket upgrade works through proxy.
- Database migrations apply cleanly.
- Logs are structured.

