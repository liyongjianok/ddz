# Deploy

部署配置目录。

当前包含：

- `docker-compose.dev.yml`：本地开发/预发一键启动栈
- `../backend/Dockerfile`：后端镜像构建文件
- `../frontend/Dockerfile`：前端镜像构建文件
- `../frontend/nginx.conf`：前端静态资源与 API/WebSocket 反向代理配置

启动命令：

```bash
docker compose -f deploy/docker-compose.dev.yml up --build
```

默认访问地址：

- 前端：http://localhost:5173
- 后端健康检查：http://localhost:8080/healthz
- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`
