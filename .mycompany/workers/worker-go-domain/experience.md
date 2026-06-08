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

## CommunityService 模式 (T-9)

5. **InstallWorker 事务模式**：使用 `s.Pool.Begin(ctx)` + `defer tx.Rollback(ctx)` + `tx.Commit(ctx)`，因为 ent 有自己的连接池，无法参与 pgx 事务，所以事务内所有操作必须用原始 SQL
6. **列表查询模式**：复杂过滤用原始 SQL（动态条件 + ILIKE + 分页），取到 ID 列表后用 ent 查询完整对象（与 playbook_service 一致）
7. **text[] 插入问题**：pgx v5 插入 `text[]` 类型时需显式 cast `$n::text[]`，nil slice 会被映射为 SQL NULL，需传 `[]string{}`
