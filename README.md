# DDZ Web Game

网页版斗地主项目。当前仓库按 `docs/zh/08-task-backlog.md` 进行分 Sprint、分模块开发。

## 当前阶段

已完成：

- `T-0001 创建仓库结构`

未实现：

- 后端 HTTP 服务。
- 认证。
- 游戏规则。
- 房间。
- WebSocket。
- 前端业务页面。

以上内容属于后续任务，不应在 `T-0001` 中实现。

## 目录结构

```text
.
├── backend/   # Go 后端
├── frontend/  # Vite React TypeScript 前端
├── deploy/    # 部署配置
└── docs/      # 研发文档
```

## 后端命令

```bash
cd backend
go test ./...
go run ./cmd/server
```

当前 `cmd/server` 仅为脚手架占位入口。真实 HTTP server 将在 `T-0002` 实现。

## 前端命令

建议使用 Node.js 18+。

```bash
cd frontend
npm install
npm run dev
npm run build
```

当前前端仅为 Vite React TypeScript 脚手架。登录、大厅和房间 UI 将在后续任务实现。

## AI 开发约束

每次只执行一个 backlog 任务。开始任何任务前先阅读：

- `docs/zh/07-ai-coding-rules.md`
- `docs/zh/08-task-backlog.md`
- 与当前任务相关的专题文档

