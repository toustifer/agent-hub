# worker-go-domain · 经验

> 业务逻辑层 Worker — 负责 5 个 service + ent 客户端集成
>
> Owner: stifer

## 任务边界

- **负责文件**：
  - `internal/hub/service/*.go`
  - `internal/hub/repository/*.go`（client.go / migrator.go 已有）
- **不负责**：handler / router / 前端 / 部署

## 已有依赖

- ent schema 已写：`ent/schema/{hub_business,hub_worker,hub_lock,hub_playbook,hub_event}.go`
- 5 张表 migration 已写：`migrations/0001-0003.sql`
- ent 客户端包装：`internal/hub/repository/client.go`（`NewClient` 入口）
- migrator：`internal/hub/repository/migrator.go`

## 关键约束

1. **LockService** 是核心，必须用 partial unique index（`INSERT ON CONFLICT DO NOTHING` + `SELECT` 验证）
2. **PlaybookService** 全文搜索必须用 SQL `to_tsvector('simple', ...)`，不用 LIKE
3. **EventService** SSE streaming 用 `chan *ent.HubEvent` + 定时 tail
4. **BusinessService** 实现白名单：默认 status='pending'，super-admin 改 'active'
