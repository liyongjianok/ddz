# DDZ Web Game

网页版斗地主项目。

当前仓库按 `docs/zh/08-task-backlog.md` 逐任务推进，已完成核心后端、WebSocket、前端 MVP、战绩持久化和玩家统计等阶段性能力。

## 目录结构

```text
.
├── backend/   # Go 后端
├── frontend/  # Vite React TypeScript 前端
├── deploy/    # 部署配置
└── docs/      # 中英文研发文档
```

## 后端

```bash
cd backend
go test ./...
go run ./cmd/server
```

默认监听 `:8080`。

健康检查：

```bash
curl http://localhost:8080/healthz
```

## 前端

建议使用 Node.js 18+。

```bash
cd frontend
npm install
npm run dev
npm run build
```

开发模式默认地址：

- 前端：http://localhost:5173
- 后端：http://localhost:8080

## Docker Compose

可以一条命令启动前端、后端、PostgreSQL 和 Redis：

```bash
docker compose -f deploy/docker-compose.dev.yml up --build
```

启动后默认地址：

- 前端：http://localhost:5173
- 后端健康检查：http://localhost:8080/healthz
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`

## 开发约束

开始任何任务前，先阅读：

- `docs/zh/07-ai-coding-rules.md`
- `docs/zh/08-task-backlog.md`
- 与当前任务相关的专题文档
