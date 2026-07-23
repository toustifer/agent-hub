# Hub ↔ agentflow 同步内容契约（Sync Contract）

> **状态：** 2026-07-22 **v0.2** — 公司协作默认同步 + 账号归因  
> **原则：** Hub 是公司协作中心；同步 = 业务对象投影（event-driven UPSERT），**不是**整库复制。  
> **源：** agentflow SQLite 为执行真相；Hub PG 为团队可见镜像。  
> **写策略：** soft-fail，永不阻塞本地任务。  
> **计划：** [docs/plans/2026-07-22-team-collab-sync-plan.md](./plans/2026-07-22-team-collab-sync-plan.md)

相关：

| 文档 | 关系 |
|------|------|
| [AGENTFLOW_ALIGNMENT.md](./AGENTFLOW_ALIGNMENT.md) | Tab / API 对齐矩阵 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Hub 表结构与鉴权 |
| 本文 | **只规定「同步什么字段」** |

---

## 0. 总原则

```text
协作问题（谁在干什么 / 卡在哪 / 谁占分支 / 共同文档 / 可复用 worker）→ 进 Hub
执行细节（worktree 路径 / BT 状态机 / 私密草稿 / 大 diff）→ 留 agentflow
```

| 规则 | 说明 |
|------|------|
| **投影，非整库** | 只 UPSERT 协作字段；不推 SQLite 文件、不推 schema |
| **事件驱动** | 状态变化时写；禁止定时全量 dump |
| **方向默认 L→H** | agentflow → Hub；H→L 拉任务为后续能力 |
| **源真相** | 任务生命周期 / 本机 worker 运行：agentflow；成员 / 审批：Hub |
| **幂等** | 同一 `business_id + 业务主键` 可重复 UPSERT |
| **soft-fail** | Hub 不可达 → 记 note，任务继续 |
| **体量上限** | 团队 doc ≤ 64KB；事件 payload ≤ 4KB；超限截断或 400 |

### 0.1 公司默认（locked v0.2）

1. **团队协作对象默认同步**：任务摘要、分支、**带账号的工作日志**、**团队 Doc**、**Worker 模板**。  
2. **每人一份 agentflow SQLite 执行库**；Hub 不是 DB 副本；进度靠投影，不靠传 `*.db`。  
3. **工作日志必须归因 Hub 账号**（`actor_user_id` + `actor_email`）+ `actor_role`；经模板执行时带 `actor_worker_id`。  
4. 同事**复用 worker 模板**（同名工种），不共享运行态（pid/lease）。  
5. **团队 Doc 更新及时**：`visibility=team` 保存即 UPSERT，带 `updated_by_*`。  
6. JWT 写入时 **服务端 stamp 用户**，客户端不得伪造他人 `user_id`。  
7. **邮箱展示（GitHub commit 级，v0.2.1）**：团队可见的人为痕迹默认展示 **邮箱**；`worker_id` 是工种不是人。见 §0.3。

### 0.2 进度模型

```text
你 SQLite  --project L→H-->  Hub 团队  <--project L→H--  同事 SQLite
```

### 0.3 何处必须标邮箱（对齐 GitHub commit author）

| 场景 | 必须露邮箱？ | 字段 |
|------|--------------|------|
| 任务生命周期动态 | ✅ | `hub_events.actor_email` |
| 团队文档 | ✅ | `created_by_email` / `updated_by_email` |
| Worker 模板发布 | ✅ | `published_by_email` |
| 任务负责人 / 最后推进人 | ✅ 建议 | 最近 event 的 email，或未来 `assignee_email` |
| 分支 reporter | ✅ 建议 | report 时 JWT email → `last_reporter` |
| 成员/邀请 | ✅ | `hub_users.email` |
| system / 纯机器 heartbeat | 否（标 system） | `actor_role=system` |

**展示默认：** `邮箱 · [role] · worker? · 动作 · 时间`  
**禁止：** JWT 人操作时团队 UI 只显示 `leader`/`worker` 而无邮箱。

---

## 1. 对象总览

| # | 对象 | 方向 | 优先级 | 状态 | Hub 表 |
|---|------|------|--------|------|--------|
| 1 | **Members / Approvals** | Hub-native | — | ✅ | memberships / invites / link_requests |
| 2 | **Branches + Bindings** | L→H | P0 | ✅ soft-report | hub_branches / hub_branch_bindings |
| 3 | **Tasks（DAG 投影）** | L→H | P0 | ✅ API；agentflow 自动写另跟 | hub_dag_state |
| 4 | **Work logs（生命周期事件）** | L→H | **P0** | ✅ 归因列 + JWT/MCP | hub_events |
| 5 | **Team docs** | L→H / UI | **P1** | ✅ | hub_team_docs |
| 6 | **Worker templates** | publish | **P1** | ✅ | hub_worker_templates |
| 7 | **Workers 在线态** | L→H | P2 | 🟡 heartbeat | hub_workers |
| 8 | **Locks** | L→H | P2 | 🟡 跨机时 | hub_locks |
| 9 | **Playbooks（经验库）** | L→H | P2 | 🟡 手动 | hub_playbooks |
| 10 | **Diary / 私密 doc 全文整表** | — | **永不** | — | 仅精选进 team docs / 短事件 |

---

## 2. 不进同步（Never-sync）

| 类别 | 例子 | 原因 |
|------|------|------|
| 整库 | SQLite 文件、schema dump、docs_sync 全树 | 无法合并、含执行私货 |
| 本地路径 | worktree 绝对路径、DBPath | 跨机无意义 |
| BT / 调度内部 | 节点态、lease 全文 | 执行器私有 |
| 海量过程体 | 未筛选 diary 全历史 | 体积与噪声 |
| 密钥 | token、api_key、未脱敏 prompt | 安全 |
| Git 实体 | 完整 diff、blob、pack | 真相在 origin |
| Worker **运行态** | pid、本机会话 | 共用模板不是进程 |

---

## 3. 工作日志归因（v0.2 强制）

### 3.1 字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `actor_user_id` | 人操作 ✅ | Hub `hub_users.id`，JWT stamp |
| `actor_email` | 人操作 ✅ | 展示 |
| `actor_role` | ✅ | `leader` \| `worker` \| `reviewer` \| `system` |
| `actor_worker_id` | 模板执行时 ✅ | 团队共用模板名 |
| `actor` | 兼容 | 优先 email |
| `event_type` | ✅ | 如 `task.started`、`doc.updated` |
| `payload` | 短 JSON | task_id, title, from, to, branch；禁路径/密钥/长文 |

### 3.2 标准 event_type

| event_type | 典型 role |
|------------|-----------|
| `task.created` | leader |
| `task.started` / `task.submitted` | worker |
| `task.passed` / `task.rework` | reviewer |
| `task.reassigned` / `task.cancelled` | leader |
| `doc.updated` / `doc.published` | worker 或 leader |
| `worker.template_published` | leader 或 worker |
| `branch.reported` | worker / system（可降噪） |

### 3.3 禁止

- 仅有 `actor=leader` 且 JWT 用户可知时不写 `actor_user_id`  
- payload 塞全文 description / diary / diff  

---

## 4. 团队 Doc

| 字段 | 说明 |
|------|------|
| `doc_key` | 业务主键（+ business_id） |
| `title` / `content` / `category` | 内容 |
| `visibility` | `team`（默认同步）\| `private`（不出现在他人 list） |
| `created_by_*` / `updated_by_*` | 账号归因 |
| `updated_at` | 及时性排序 |

触发：Dashboard / MCP / agentflow 显式 upsert；team 可见保存即投影。

---

## 5. Worker 模板

| 字段 | 说明 |
|------|------|
| `worker_id` | 模板名（同事安装用同一 id） |
| `display_name` / `description` / `skills` | 说明 |
| `handbook` | jsonb 摘要 |
| `prompt_summary` | 脱敏短文本，非密钥 prompt 全文 |
| `published_by_*` | 发布人账号 |
| `visibility` | 默认 `team` |

**不同步：** 运行态 pid/host/lease（可仍走 `hub_workers` heartbeat，与模板表分离）。

---

## 6. Tasks / Branches（摘要，细节见历史章节）

- **Tasks：** `hub_dag_state` UPSERT：task_id, title, status, assigned_worker, depends_on, output_files, branch, head_sha  
- **Branches：** report tips + bindings（dag\|task\|worker\|user）；不报绝对路径  

Status 建议与 agentflow 对齐：`assigned` / `executing` / `review_pending` / `rework_needed` / `done` / `cancelled`。

---

## 7. API 速查（v0.2）

```text
POST /v1/hub/events
GET  /v1/hub/events?business=

POST /v1/hub/businesses/:code/docs
GET  /v1/hub/businesses/:code/docs
GET  /v1/hub/businesses/:code/docs/:key

POST /v1/hub/businesses/:code/worker-templates
GET  /v1/hub/businesses/:code/worker-templates
GET  /v1/hub/businesses/:code/worker-templates/:worker_id
```

MCP：`hub_append_event`、`hub_list_events`、`hub_upsert_doc`、`hub_list_docs`、`hub_get_doc`、`hub_publish_worker_template`、`hub_list_worker_templates`。

---

## 8. agentflow 接线（接口约定，实现另仓）

1. task create/transition → dag sync + events（role/worker 从上下文）  
2. 每人本 JWT + `business_code`  
3. 未来：H→L 订阅任务列表（仍非传 db）  
