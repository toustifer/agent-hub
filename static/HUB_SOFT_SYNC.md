# agentflow → Hub soft-sync（一点击写什么）

> 状态：Phase E 已接线（2026-07-23）  
> Hub 基址：`https://hub.stifer.xyz`  
> 契约：agent-hub `docs/SYNC_CONTRACT.md` v0.2  

## 配置（每人自己的 JWT）

`~/.agent-hub/config.json` 或 `{workdir}/.mycompany/hub-client.json`：

```json
{
  "hub_url": "https://hub.stifer.xyz",
  "token": "<Hub JWT，Dashboard 登录或 device login>",
  "business_code": "zhiji",
  "email": "you@example.com"
}
```

| 项 | 说明 |
|----|------|
| `token` | **必须**人侧 JWT，Hub 用它 stamp `actor_email` |
| `business_code` | 团队 code（如 `zhiji`） |
| `email` | 可选；分支 report 默认 reporter |
| 关闭同步 | `HUB_SYNC=0` 或去掉 token/business |

## 何时自动写 Hub

本地 **成功** 后 soft 调用（失败只记 note，不挡任务）：

| 本机动作 | Hub 写入 |
|----------|----------|
| `task_create` / batch | `hub_dag_state` UPSERT + event `task.created` (role=leader) |
| prepare_start / start | dag UPSERT + `task.started` (role=worker) + 可能 branch report |
| submit | dag + branch report + `task.submitted` |
| pass | dag + `task.passed` (reviewer) |
| rework | dag + `task.rework` (reviewer) |
| reassign | dag + `task.reassigned` (leader) |
| cancel | dag + `task.cancelled` (leader) |
| resume | dag + `task.resumed` (worker) |

响应字段：`hub_task_sync`（含 `task_sync` 与 `event_append` note）。

## 账号怎么出现

- **邮箱 / user_id**：Hub 从 JWT stamp（客户端不伪造）  
- **actor_role**：agentflow 按 transition 映射  
- **actor_worker_id**：任务的 `AssignedWorker`（模板名，可复用）  

查询：Hub MCP `hub_list_events` / `hub_get_dag`（含 `last_by=` / `last_actor_email`）。

## 不会自动写的

- 团队 Doc 全文（用 Hub `hub_upsert_doc`）  
- Worker 模板发布（用 `hub_publish_worker_template`）  
- 本机 SQLite / 路径 / BT 状态  

## 联调检查清单

1. 重编 agentflow 二进制并重启 MCP  
2. config 有 token + business_code  
3. 本地 create 一任务 → start  
4. Hub：`hub_list_events` 见 `task.created` / `task.started` + 你的邮箱  
5. Hub：`hub_get_dag` 见任务，且 `last_by=你的邮箱`  

## 代码入口

- `pkg/hub/event.go` — `AppendEvent`  
- `pkg/hub/task.go` — `SyncTask`  
- `pkg/server/hub_sync.go` — `softHubAfterTask` / `mapTaskLifecycle`  
- call sites：`mcp.go` create/transition、`handlers_ext.go` batch create  
