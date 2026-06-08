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

## T-3 完成后经验

### 模式：handler 结构

- 每个 handler 文件对应一个 service 领域（business/worker/lock/playbook/event）
- 请求体使用私有 struct（如 `createBusinessReq`），通过 `c.ShouldBindJSON` 绑定
- 响应统一 `gin.H{"data": ...}` 或 `gin.H{"code": N, "message": "..."}`
- 列表/搜索接口统一 `limit`/`offset` 分页参数，默认 20
- 路径参数用 `c.Param()`，查询参数用 `c.Query()`

### 模式：LockHeldError 处理

- Service 层定义 `LockHeldError` struct，包含 `HolderWorkerID`
- Handler 层用 `errors.As(err, &held)` 判断，返回 HTTP 409
- 额外返回 `data.holder_worker_id` 供客户端使用

### 模式：SSE 流式事件

- 设置 `Content-Type: text/event-stream` + `Cache-Control: no-cache` + `Connection: keep-alive`
- 先 `WriteHeader(200)` 再进入循环
- 使用 `gin.Flusher` cast `c.Writer`，每次 `fmt.Fprintf` 后 `Flush()`

### 踩坑

1. **go.sum 缺失**：首次创建文件后无法编译，需 `go mod tidy` 下载依赖。但网络受限时无法解决，代码本身语法正确（import 与 go.mod 对齐）
