# 实施计划

> 起始日期：2026-06-05
> 当前阶段：Phase 1 进行中
> 目标：1.5 周内完成 Phase 0-6，POC 跑通

## 决策历史

| 日期 | 决策 | 理由 |
|---|---|---|
| 2026-06-05 | 路径 2（独立服务 + go.mod replace 复用） | 避免改 sub2api 源码，零生产风险 |
| 2026-06-05 | 共享 PostgreSQL + `hub` schema | 不重搭 DB，sub2api 表完整保留 |
| 2026-06-05 | 复用 sub2api 认证 | 不重新实现 JWT/APIKey，耦合面最小 |
| 2026-06-05 | 端口 9000 | 避开 8080/8000/5173/5432/6379/20241 |
| 2026-06-05 | 独立 Vue 3 SPA | 不嵌入 sub2api 静态，前端独立部署 |
| 2026-06-05 | 锁用 partial unique index | PG 原生支持，无需 advisory lock |
| 2026-06-05 | tsvector 全文搜索 | 复用 PG 能力，比 FTS5 简单 |
| 2026-06-05 | append-only event 表 | 审计可追溯，SSE 推流 |
| 2026-06-05 | hub 自带 TimeMixin / SoftDeleteMixin | 不 import sub2api 内部包，更解耦 |
| 2026-06-05 | 前端作为必做项 | 用户明确需要 Dashboard |

## Phase 0：项目骨架（0.5 天）✅

- [x] D:\myprogram\agent-hub 目录创建
- [x] git init + initial commit (5aa9977)
- [x] 目录结构搭建
- [x] README.md
- [x] go.mod（含 replace 指令）
- [x] .gitignore
- [x] .env.example
- [x] docs/ARCHITECTURE.md
- [x] docs/PLAN.md
- [x] docs/INTEGRATION.md
- [x] docs/OPERATIONS.md
- [x] cmd/hub/main.go 占位
- [x] ent/generate.go 占位
- [x] internal/config/config.go 占位
- [x] internal/server/server.go（含 /health）
- [ ] go mod download 验证（Phase 6 在服务器上验证）

## Phase 1：数据层（2 天）🔄 进行中

### 1.1 ent mixin
- [x] ent/schema/time_mixin.go（hub 自有）
- [x] ent/schema/soft_delete_mixin.go（hub 简化版，字段 + 手动 WHERE）

### 1.2 ent schema
- [x] ent/schema/hub_business.go
- [x] ent/schema/hub_worker.go
- [x] ent/schema/hub_lock.go
- [x] ent/schema/hub_playbook.go
- [x] ent/schema/hub_event.go
- [ ] go generate ./ent 验证（需 Go 1.26+ + sub2api 源码就位）

### 1.3 索引 migration
- [x] migrations/0001_init_schema.sql（5 张表 + 普通索引）
- [x] migrations/0002_partial_unique_locks.sql（核心：partial unique index）
- [x] migrations/0003_tsvector_playbook.sql（trigger + GIN 索引）
- [x] internal/hub/repository/migrator.go（按文件名排序 + 增量执行）
- [x] 复制 SQL 到 internal/hub/repository/migrations/（embed 用）

### 1.4 后续
- [ ] repository 层 skeleton（ent.Client 包装）
- [ ] 写一个 mock 测试验证 migrator 跑通

## Phase 2：服务层（2 天）

### 2.1 HubBusinessService
- [ ] Create / Get / List / Update / Delete

### 2.2 HubWorkerService
- [ ] Register / Heartbeat / List / CleanupDead

### 2.3 HubLockService（核心）
- [ ] Acquire（partial unique index + ON CONFLICT）
- [ ] Renew
- [ ] Release
- [ ] CleanupExpired（cron）
- [ ] ListActive

### 2.4 HubPlaybookService
- [ ] Upload
- [ ] Search（tsvector）
- [ ] Get / List

### 2.5 HubEventService
- [ ] Append
- [ ] List
- [ ] Stream（SSE）

## Phase 3：HTTP 层（1 天）

- [ ] 复用 sub2api auth_service 封装 middleware
- [ ] 5 个 handler
- [ ] router.go
- [ ] main.go 启动

## Phase 4：Hub Dashboard 前端（1-2 天）★ 必做

- [ ] Vite + Vue 3 + TS 初始化
- [ ] 路由 + 菜单
- [ ] 业务列表 / 注册
- [ ] Worker 心跳列表
- [ ] 活跃锁列表
- [ ] Playbook 搜索
- [ ] 事件流（SSE）

## Phase 5：hub-client SDK（1-2 天）

- [ ] hub-client.js（Node.js）
- [ ] hub-client.py（Python）
- [ ] 接入示例

## Phase 6：部署（1 天）

- [ ] go build 验证
- [ ] systemd unit
- [ ] nginx 配置
- [ ] cloudflared 加 hub.stifer.xyz
- [ ] 冒烟测试

## Phase 7：跨业务接通（1 周）

- [ ] siruoning 接入
- [ ] insight-tutor 接入
- [ ] 压测
- [ ] 文档完善

## 验证标准（POC 完成）

- [ ] `sub2api` 二进制 0 改动
- [ ] `agent-hub` 独立编译运行
- [ ] `https://hub.stifer.xyz/health` 返回 200
- [ ] 5 张 hub 表已建
- [ ] 浏览器开 `https://hub.stifer.xyz/` 看到 dashboard
- [ ] 2 个模拟 Worker 抢锁互斥成功
- [ ] siruoning hub-boot.js 跑起来，dashboard 看到心跳
- [ ] playbook 搜索能命中
