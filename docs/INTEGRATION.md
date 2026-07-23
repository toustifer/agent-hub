# 业务接入指南

> 面向：要在自己业务仓里接入 agent-hub 的开发者

## 0. 身份与登录（团队就绪）

**主路径是用户登录 JWT（Web 或 MCP `hub_login`），不是 API Key。**  
建团后操作团队资源（invite / branch report / dag / repo）需要：**已登录 + 该 business 的 membership**。

### Web / CLI 密码登录

```bash
# CLI（写入 ~/.agent-hub/config.json）
node cli/login.js
# 提示 Email + Password → POST /v1/hub/auth/login { email, password }
```

注册需邮箱验证（QQ SMTP）。未验证邮箱不能登录。

### MCP Device 登录（Claude）

1. `hub_login()` → 得到浏览器 URL + code  
2. 浏览器打开 URL，**先用真实用户登录**，再点 Approve（confirm 需 Bearer JWT，`uid>0`）  
3. `hub_login({ code })` → token 写入 `~/.agent-hub/config.json`，重启可复用  
4. `hub_list_my_businesses` 验证

**安全底线：** 发码时 device code 的 token 为空；confirm 时按登录用户 **重签 JWT**。旧的 `uid=0` token 一律 401。

### 创建 Business

```bash
curl -X POST https://hub.stifer.xyz/v1/hub/businesses \
  -H "Authorization: Bearer $HUB_USER_JWT" \
  -H "Content-Type: application/json" \
  -d '{"code":"my-team","name":"My Team","description":"..."}'
```

创建者 membership 角色为 **`owner`**。**默认不生成 API Key**；人侧用 Web 登录 JWT 或 MCP `hub_login` 操作。

```json
{
  "data": {
    "business": { "...": "..." },
    "role": "owner",
    "note": "use Web login or MCP hub_login (JWT) to manage this team; API key is optional for CI/workers"
  }
}
```

本地连接：

| 用途 | 配置 |
|------|------|
| 人侧 JWT | MCP `hub_login` → `~/.agent-hub/config.json` 的 `token`；或 env `HUB_TOKEN` / `HUB_JWT` |
| 业务 code | env `HUB_BUSINESS_CODE`，或 config / `.mycompany/hub-client.json` 的 `business_code` |
| 机器 Key | **可选**，仅 CI/心跳/锁；见文末附录 |

**禁止** 使用历史魔法串 `agent-company-worker` / 默认 `ai-medbox`。

运维若要在创建时仍自动发一把机器 Key：服务端设 `HUB_CREATE_API_KEY=1`（一次性明文仍只返回一次）。

## 1. 加入策略 join_policy

| 值 | 行为 |
|----|------|
| `invite_only`（默认） | 仅 invite token 或 link-request 审批可加入 |
| `approval` | 禁止 open join；走 link-request |
| `open` | 任意登录用户可 `POST .../join` |

```sql
UPDATE hub.hub_businesses SET join_policy = 'open' WHERE code = 'demo';
```

### 邀请成员（无 SMTP：复制 URL）

```bash
curl -X POST https://hub.stifer.xyz/v1/hub/businesses/my-team/invite \
  -H "Authorization: Bearer $JWT" \
  -d '{"email":"b@example.com","role":"member"}'
# → invite_url = https://hub.stifer.xyz/invite/accept?token=<raw once>
```

要求：**调用者必须是该 team 的 admin/owner**（membership 门禁）。

对方：注册/登录（邮箱须匹配）→ 打开 invite_url 或 `POST /v1/hub/invites/accept` `{token}`。

MCP：`hub_invite_member` / `hub_accept_invitation({token})`（JWT only）。

### Link-request

- 申请：`POST /v1/hub/businesses/:code/link-requests`  
- 列表/审批：admin|owner only  

## 2. 项目配置

人侧最小配置（**无需 api_key**）：

```json
{
  "hub_url": "https://hub.stifer.xyz",
  "business_code": "my-team",
  "token": "<from hub_login / Web login>"
}
```

写在：

- `~/.agent-hub/config.json`（MCP 登录默认落盘），或  
- `/path/to/your-business/.mycompany/hub-client.json`（gitignore）

机器 worker 额外字段（可选）：

```json
{
  "hub_url": "https://hub.stifer.xyz",
  "business_code": "my-team",
  "worker_id": "worker-medication",
  "api_key": "<machine key, optional>",
  "heartbeat_interval_seconds": 30,
  "lock_default_ttl_seconds": 300
}
```

`.gitignore`：

```
.mycompany/hub-client.json
```

## 3. MCP 工具（stdio = HTTP，单源 lib/tools.js）

| 工具 | 鉴权 | 备注 |
|------|------|------|
| hub_login | 无 | device 流；落盘 token |
| hub_list_my_businesses | JWT | 不要求 business_code |
| hub_invite_member / accept / link-* | JWT + membership | 团队门禁 |
| hub_list_* / dag / repos / branches | JWT + membership | 管理面；无 token 报「先 hub_login」 |
| hub_report_branches / hub_bind_branch / hub_refresh_branches | JWT + membership | 人侧主路径；机器也可用 API Key |
| hub_heartbeat / locks / events / playbooks | **机器 API Key** | 无人值守；缺 key 明确提示机器路径 |

```bash
cd mcp-server && npm run stdio   # Claude
npm run http                     # :9001
```

## 3b. Branch 索引（同步 GitHub / 本机 tip）

**契约：** 代码分 fork 在 git branch；过程库（docs/tasks）**不按 branch 分库**。Hub 只存 branch 镜像与绑定，不存 doc 全文。

所有带 `:code` 的接口：JWT 用户须为该 business 成员；API Key 须与 path code 匹配。

| Method | Path | 谁 |
|--------|------|-----|
| POST | `/v1/hub/repos/:code` | JWT + member | 绑 remote + default_branch |
| POST | `/v1/hub/repos/:code/branches/report` | JWT + member **或** API Key | 批量 tip + bindings |
| GET | `/v1/hub/repos/:code/branches` | JWT + member **或** API Key | 列表 tip + 占用 |
| POST | `/v1/hub/repos/:code/branches/bind` | JWT + member **或** API Key | task/dag/worker 占用 |
| POST | `/v1/hub/repos/:code/branches/unbind` | JWT + member **或** API Key | 释放绑定 |
| POST | `/v1/hub/repos/:code/branches/refresh` | JWT + member | 从 GitHub API 或 ls-remote 刷新 tip/PR |

```bash
# 人侧上报（推荐：Bearer JWT，无 api_key）
curl -X POST "https://hub.stifer.xyz/v1/hub/repos/$HUB_BUSINESS_CODE/branches/report" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"branches":[{"name":"feature/pay","tip_sha":"abc…","source":"report"}],
       "bindings":[{"bind_type":"task","bind_id":"T12","branch_name":"feature/pay","head_sha":"abc…"}]}'

# 全队查看
curl "https://hub.stifer.xyz/v1/hub/repos/$HUB_BUSINESS_CODE/branches" \
  -H "Authorization: Bearer $JWT"

# GitHub / origin tip 刷新
curl -X POST "https://hub.stifer.xyz/v1/hub/repos/$HUB_BUSINESS_CODE/branches/refresh" \
  -H "Authorization: Bearer $JWT" -H "Content-Type: application/json" -d '{}'
```

agentflow 自动 report（prepare/submit soft-fail）：

1. env：`HUB_TOKEN`|`HUB_JWT` + `HUB_BUSINESS_CODE`（+ 可选 `HUB_BASE_URL`）  
2. `{workdir}/.mycompany/hub-client.json`：`token` + `business_code`  
3. `~/.agent-hub/config.json`：MCP 登录的 `token` + 可选 `business_code`  
4. 仅有 `api_key` 仍兼容机器路径  

无 token/key + business_code → `hub_report_skipped: no login token / business_code`。

Hub 服务端 `GITHUB_TOKEN` / `HUB_GITHUB_TOKEN`：refresh 走 GitHub API；否则尝试 `git ls-remote`。

## 4. Worker 心跳 / 锁（机器路径）

Header：

```
X-API-Key: <real>
X-Business-Code: <code>
```

```js
const HubClient = require('@stifer/hub-client')
const config = require('./hub-client.json')
const hub = new HubClient(config)
hub.heartbeat()
setInterval(() => hub.heartbeat(), config.heartbeat_interval_seconds * 1000)

async function performMedicationDeletion(id) {
  return await hub.withLock(
    'siruoning.medication.pages.Homepage',
    async () => { /* ... */ },
    { ttl: 300 }
  )
}
```

人侧协作**不必**配 API Key。机器 Key 获取见附录。

## 5. Dashboard

`https://hub.stifer.xyz/` — 创建团队成功后：进入团队 / 用 MCP `hub_login` 操作（**不再弹「只显示一次」api_key**）。

Team 页：

- **Branches**：tip 短 SHA、source、PR 链接、bindings（stale 标红）；可 **Refresh from GitHub / origin**
- **Members**：邀请、pending invites、成员列表
- **Approvals**：link-request

## 6. 故障排查

| 现象 | 原因 | 处理 |
|------|------|------|
| device confirm 401 | 未登录 / uid=0 旧 token | 重新登录再批准 |
| join 409 invite-only | 默认门禁 | 用 invite token 或 link-request |
| branch/report 403 | 非成员 JWT 或 key 与 code 不匹配 | 先 invite/accept，或检查 business_code |
| MCP 报 Not logged in | 无 token | 先 `hub_login()`，**不要**去抄 api_key |
| heartbeat 401 | 无机器 key / 魔法 key | 附录生成机器 Key；人侧心跳可跳过 |
| invite accept email mismatch | 登录邮箱 ≠ 邀请邮箱 | 用被邀邮箱注册登录 |

## 7. 与 agentflow 边界

> **完整对齐矩阵（团队页每个 Tab × 谁写 × 缺口 × 路线图）：**  
> **[AGENTFLOW_ALIGNMENT.md](./AGENTFLOW_ALIGNMENT.md)** · 线上 https://hub.stifer.xyz/agentflow-alignment.md

摘要：

- **Branch 图**：Hub 共享；agentflow prepare/start/submit **soft report**（JWT 优先）— **已齐**  
- **任务 DAG / Workers / 锁 / 经验库 / 事件**：Hub 有表与 MCP；agentflow **大多不自动写** — 见对齐文档 §3、§8  
- **过程库**：agentflow 统一 namespace；**不**按 branch 拆 SQLite  
- 本地 `docs_sync` 只镜像到 `.mycompany/agentflow/`，**不上 Hub**  
- Approvals / Members：**纯人侧**，与 agentflow review 无关  

## 8. Playbook / 搜索 / 升级

```js
await hub.playbookUpload({
  category: 'decisions',
  title: '...',
  content: '## 背景\n...',
  tags: ['medication']
})
const results = await hub.playbookSearch('回滚', { business: 'my-team' })
```

agent-hub 升级时，hub-client SDK 可能要同步升；破坏性升级会发警告。

---

## 附录：机器 API Key（可选 / CI）

人侧主路径**不需要** Key。仅当无人值守 worker（heartbeat、分布式锁、CI 写 branch）时再配。

当前 MVP：

- 已有团队若创建时生成过 Key，**旧 key 仍有效**
- 运维可在服务端设 `HUB_CREATE_API_KEY=1` 后新建团队会再返回一次明文
- 后续可加：`POST /v1/hub/businesses/:code/api-keys`（owner/admin）与 revoke（Phase 5）

配置示例：

```bash
export HUB_API_KEY=...
export HUB_BUSINESS_CODE=my-team
# 或写入 ~/.agent-hub/config.json / .mycompany/hub-client.json 的 api_key
```

MCP 机器工具：`hub_heartbeat` / `hub_acquire_lock` / `hub_append_event` / `hub_create_playbook` — 缺 key 时报机器路径错误，不会引导人去「抄建团 Key」。
