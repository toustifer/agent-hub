# worker-sdk · 经验

> 多语言客户端 SDK Worker — 负责 Node.js + Python 双语言 SDK
>
> Owner: stifer

## 任务边界

- **负责文件**：
  - `sdk/hub-client.js`（Node.js）
  - `sdk/hub_client.py`（Python）
  - `sdk/README.md`（文档）
- **不负责**：Go 后端 / 前端 / 部署

## 已有依赖

- 后端 API 在 hub.stifer.xyz/v1/hub/*
- API 结构见 internal/hub/handler/*.go 和 internal/hub/router.go

## 关键约束

1. **零外部依赖**：Node.js 仅用 http/https/crypto/url 内置模块；Python 仅用 urllib.request/json/uuid 标准库
2. **双认证**：同时支持 JWT `Authorization: Bearer` 和 APIKey `X-API-Key` + `X-Business-Code`
3. **协议对齐**：自动注入 `business_code`；响应自动解包 `data` 字段
4. **错误语义**：409 锁冲突在 Python 中映射为 `HubLockError` 异常；Node.js 中为 `err.code === 409` 的 Error
5. **命名风格**：Node.js 用 camelCase（acquireLock），Python 用 snake_case（acquire_lock）

## T-5 完成后经验

### 模式：SDK 方法结构

- 构造函数接受 `baseUrl`、`apiKey`、`businessCode`、`token` 四个参数
- 所有 worker 端方法自动将 `business_code` 注入请求体
- 内部 `_request` 方法统一处理 HTTP 请求、JSON 序列化、认证头注入、错误映射
- 响应自动从 `{ data: ... }` 包裹中解包

### 模式：双认证处理

- 如果 `token` 存在 → 发送 `Authorization: Bearer` 头（用于 admin 端）
- 如果 `apiKey` 存在 → 发送 `X-API-Key` 头 + `X-Business-Code` 头（用于 worker 端）
- 两者可同时设置，三种头同时发送

### 模式：锁冲突错误处理

- 后端在锁冲突时返回 HTTP 409 + `{ code: 409, message: "...", data: { holder_worker_id: "..." } }`
- Node.js：抛出 `Error`，其 `code === 409`，`response` 属性含完整响应
- Python：抛出 `HubLockError` 异常，含 `resource_key` 和 `holder_worker_id` 属性

### 踩坑

1. **X-API-Key 大小写**：中间件读取 `X-API-Key`（非 X-Api-Key），拼写必须精确匹配
2. **响应解包**：Go gin 返回 `gin.H{"data": ...}` 统一包裹，SDK 需在 2xx 时提取 `data` 字段
3. **Python HTTPError body**：`urllib.error.HTTPError` 抛出后 body 仍在流中，需 `e.read()` 读取
