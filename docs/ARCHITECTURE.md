# 架构

## 核心原则

### P1：sub2api 0 修改
agent-hub 通过 `go.mod replace` 复用 sub2api 的 Go 包作为库，但**不修改 sub2api 任何源代码**。

理由：
- sub2api 是生产 AI 网关，挂了所有 Claude Code 调用挂
- 5GB 源码、复杂 Wire DI、ent 代码生成，改一处风险高
- 升级 sub2api 时不破坏 agent-hub

### P2：共享数据库，独立 schema
agent-hub 用同一个 PostgreSQL 实例，但所有表在 `hub` schema 下，不污染 `public` schema。

```sql
CREATE SCHEMA IF NOT EXISTS hub;
```

Auth / 团队相关表（migrations `0006`+）：

```text
hub.hub_users
hub.hub_memberships      -- role: owner | admin | member
hub.hub_api_keys         -- key_hash, revoked_at
hub.hub_device_codes     -- token NULL until confirm; user_id bound on confirm
hub.hub_invites          -- pending email invites, token_hash, 7d expiry
hub.hub_businesses.join_policy  -- invite_only | approval | open
```

业务核心表：

```text
hub.hub_businesses
hub.hub_workers
hub.hub_locks
hub.hub_playbooks
hub.hub_events
hub.hub_link_requests
hub.hub_repos / hub_branches / hub_branch_bindings   -- branch 索引 (0008)
hub.hub_dag_state                                    -- 任务摘要镜像 (+ branch/head_sha)
```

### P3：共享 Redis
sub2api 已经在用宿主机 redis（127.0.0.1:6379），agent-hub 复用同一实例：
- 锁 TTL 倒计时
- 限流
- 事件 SSE 推送队列

### P4：认证（JWT / membership 主路径；API Key 可选机器凭证）

**用户（Web / MCP 人侧工具）— 主路径：**

- `POST /v1/hub/auth/register|login` → JWT HS256，claims `uid`（必须 >0）、`sub`=email；注册需邮箱验证  
- Device 流：`POST /auth/device` 发码（token 空）→ 已登录用户 `POST /auth/device/confirm`（JWT 组）重签 token → poll `/auth/device/token`  
- 中间件拒绝 `uid<=0`；API key 回退路径 **不** 设置 `user_id`  
- 带 `:code` 的团队读写：`RequireMembership` — JWT 用户须在 `hub_memberships`；机器 key 须与 path code 匹配  
- **创建 business 默认不生成 API Key**；仅当 `HUB_CREATE_API_KEY=1` 时生成并只返回一次明文

**Worker（无人值守）— 可选：**

- Header `X-API-Key` + `X-Business-Code`  
- 校验 `hub_api_keys.key_hash`（及 `revoked_at IS NULL`）  
- 用于 heartbeat / 锁 / CI；人侧协作不依赖 Key

**加入团队：**

- 默认 `join_policy=invite_only`  
- Invite：写 `hub_invites`，返回 raw token URL；accept 校验 email 匹配；仅 admin|owner 可邀请  
- Link-request：admin|owner 审批后 membership  

### P5：MCP 远程主入口（2026-07）

```text
internal/mcp/                 # Go Streamable HTTP（官方 go-sdk）@ /mcp
  auth.go / server.go / tools.go  # 24 工具 → service/SQL；JWT + API Key
router.go                     # gin.WrapH(mcpHub.HTTPHandler())；不再反代 :9001

OAuth:
  /.well-known/oauth-authorization-server
  POST /v1/hub/oauth/register  → hub.hub_oauth_clients
  device authorize/token + authorize redirect

legacy（兼容）:
  mcp-server/index.js         # 本机 stdio 薄壳
  mcp-server/http-server.js   # 废弃的 :9001 JSON-RPC
```

Claude 主配置：`{ "type": "http", "url": "https://hub.stifer.xyz/mcp" }`。详见 `docs/MCP_FOR_AI.md` / `MCP_OFFICIAL.md`。

### P6：与 agentflow 的职责切分

- **agentflow**：本机执行源（SQLite DAG/task、worktree、BT）。  
- **Hub**：团队身份 + 跨机**索引/镜像**（branch、dag 摘要、workers、locks、playbooks、events）。  
- 当前唯一 agentflow→Hub **自动**写路径：branch soft-report。  
- **同步内容白名单（先定字段再谈 UI）：** **[SYNC_CONTRACT.md](./SYNC_CONTRACT.md)** · `/sync-contract.md`  
- 完整 Tab 对齐矩阵、缺口与路线图：**[AGENTFLOW_ALIGNMENT.md](./AGENTFLOW_ALIGNMENT.md)**。

无硬编码 business/key；人侧缺 token → 提示 `hub_login`；机器工具缺 key 才报机器路径错误。

### P6：Branch 索引 vs 统一过程库

协作语义像 **git**：代码分叉在 branch；过程状态（docs/tasks/diaries）**不按 branch 分库**。

```text
GitHub / origin          → 源码真相（各人 worktree）
Hub hub_branches         → 团队可见的 tip_sha / PR / 谁占用哪条线
Hub hub_branch_bindings  → dag|task|worker|user → branch
agentflow SQLite         → 统一过程库（doc/task/diary，一份 namespace）
task.metadata            → code_ref: git.branch, git.head_sha, worktree…
docs_sync 文件镜像       → 可选备份，非团队同步主路径
```

| 层 | 分不分 branch |
|----|----------------|
| git 源码 | 是 |
| Hub branch 索引 | 是（索引，不是 doc 体） |
| 过程库 docs/tasks | **否**，统一 |
| `.mycompany/agentflow` git commit | **不强制** |

Tip 权威：`source=github_api|ls_remote` 覆盖 `report`；bindings 以 worker report 为准。  
过程写入：doc `expected_version` CAS（短冲突），长路径锁只用于代码路径。  
docs_sync MANIFEST 可选带 `content_tree_hash`（备份对比用，非团队主路径）。  
GitHub 刷新：`POST /v1/hub/repos/:code/branches/refresh`（`GITHUB_TOKEN` 或 `git ls-remote`）写 tip + PR 字段，`source=github_api|ls_remote`。

## 5 个核心表（多租户）

### hub_businesses — 业务租户表
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| code | varchar(64) unique | 租户代号（siruoning） |
| name | varchar(128) | 租户名 |
| repo_url | text | 仓库 URL |
| owner_user_id | bigint | 创建者（CreateBusiness 传入真实 user_id） |
| description | text | 业务说明 |
| status | varchar(20) | active / suspended / pending |
| join_policy | varchar(20) | invite_only / approval / open |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### hub_workers — Worker 心跳
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| business_id | bigint FK | 关联 business |
| worker_id | varchar(128) | 业务内的 worker 名（medication） |
| version | varchar(32) | agent-company 版本 |
| status | varchar(20) | online / offline |
| last_heartbeat_at | timestamptz | |
| host / pid / owner | | 运行环境元数据 |

### Device / Invite 生命周期（摘要）

```text
DeviceAuth → INSERT code (token NULL)
  → user logs in Web → DeviceConfirm (JWT) → UPDATE token=genToken(uid), confirmed
  → MCP polls DeviceToken → returns JWT, DELETE code

InviteMember → INSERT hub_invites (token_hash), return raw once in invite_url
  → AcceptInvite → membership + status=accepted (email must match)
```

### 与 agentflow

本地 docs_sync 与 Hub 身份正交；**HubSyncer 文档推拉不在本轮实现**。
