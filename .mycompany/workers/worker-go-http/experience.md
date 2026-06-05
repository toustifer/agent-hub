# worker-go-http · 经验

> HTTP 层 Worker — 负责 5 个 handler + middleware + router
>
> Owner: stifer

## 任务边界

- **负责文件**：
  - `internal/hub/handler/*.go`
  - `internal/hub/router.go`
  - `internal/middleware/*.go`
  - `internal/server/server.go`（需要更新以使用新 router）
- **不负责**：service 实现 / 前端 / 部署

## 已有依赖

- 5 个 service 由 worker-go-domain 写
- ent 客户端由 repository/client.go 提供
- 5 个 ent schema 已定义

## 关键约束

1. **JWT middleware** 复用 sub2api `internal/service.AuthService.ValidateToken`（通过 go.mod replace 拿到）
2. **APIKey middleware** 复用 sub2api `internal/service.APIKeyService.ValidateKey`
3. **路由**：
   - `/v1/hub/businesses` 等 admin 端用 JWT middleware
   - `/v1/hub/workers/heartbeat` 等 worker 端用 APIKey middleware
4. **错误返回**：统一 `{code, message}` JSON
5. **CORS** 配置从 .env 读取
