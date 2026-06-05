# 业务接入指南

> 面向：要在自己业务仓里接入 agent-hub 的开发者

## 1. 业务注册（一次性）

找 hub admin 给你注册业务：

```bash
curl -X POST https://hub.stifer.xyz/v1/hub/businesses \
  -H "Authorization: Bearer $HUB_ADMIN_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "siruoning",
    "name": "AI 智能药盒",
    "repo_url": "git@github.com:stifer/siruoning.git",
    "description": "微信小程序 + 后端 + 硬件"
  }'
```

返回 `business_id`，记下来。

## 2. 创建 APIKey（每个 worker 一个）

```bash
curl -X POST https://hub.stifer.xyz/v1/hub/apikeys \
  -H "Authorization: Bearer $HUB_ADMIN_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "business_code": "siruoning",
    "name": "worker-medication",
    "scope": "hub:write"
  }'
```

返回 `api_key`，形如 `hub_siruoning_med_xxxxxxxxxxxx`，**妥善保管**。

## 3. 在业务仓里配置

`/path/to/your-business/.mycompany/hub-client.json`：

```json
{
  "hub_url": "https://hub.stifer.xyz",
  "business_code": "siruoning",
  "worker_id": "worker-medication",
  "api_key": "hub_siruoning_med_xxxxxxxxxxxx",
  "heartbeat_interval_seconds": 30,
  "lock_default_ttl_seconds": 300,
  "auto_upload_decisions": true
}
```

`.gitignore` 加：
```
.mycompany/hub-client.json
```

## 4. 启动 hub-boot

`/path/to/your-business/.mycompany/hub-boot.js`：

```js
const HubClient = require('@stifer/hub-client')
const config = require('./hub-client.json')

const hub = new HubClient(config)

// 启动时立即心跳
hub.heartbeat()

// 周期心跳
setInterval(() => hub.heartbeat(), config.heartbeat_interval_seconds * 1000)

// Worker 启动时通知
hub.eventEmit('worker.started', {
  pid: process.pid,
  started_at: new Date().toISOString()
})

// 进程退出时通知
process.on('SIGINT', async () => {
  await hub.eventEmit('worker.shutdown', { reason: 'SIGINT' })
  process.exit(0)
})
```

启动：
```bash
node .mycompany/hub-boot.js &
```

## 5. 业务代码里用锁

`medication/pages/Homepage/Homepage.js` 删药品前：

```js
const HubClient = require('@stifer/hub-client')
const config = require('./hub-client.json')
const hub = new HubClient(config)

async function performMedicationDeletion(id) {
  return await hub.withLock(
    'siruoning.medication.pages.Homepage',
    async () => {
      // 原有逻辑
      const currentPlans = await HabitDataService.getUserPrescriptionTemplate()
      // ... 乐观更新
    },
    { ttl: 300 }  // 5 分钟自动过期
  )
}
```

冲突时会抛 `HubLockConflictError`，可以：
- `await sleepAndRetry(5000)` 重试
- 看 dashboard 谁在占用
- 强制抢（admin 操作）

## 6. 上传 playbook

新写了一条决策 / 踩坑：

```js
await hub.playbookUpload({
  category: 'decisions',
  title: '药品删除必须 snapshot 频率设置',
  content: '## 背景\n...',
  tags: ['medication', 'rollback', 'optimistic-update']
})
```

或者 CLI：
```bash
hub-cli playbook upload \
  --category decisions \
  --title "..." \
  --content-file ./notes.md \
  --tags "medication,rollback"
```

## 7. 搜索 playbook

跨业务搜索：
```js
const results = await hub.playbookSearch('回滚')
// 命中所有业务的决策
```

限定业务：
```js
const results = await hub.playbookSearch('回滚', { business: 'siruoning' })
```

## 8. 看 dashboard

`https://hub.stifer.xyz/` 用 admin 账号登录后能看到：
- 业务列表
- 每个 business 的 Worker 心跳
- 活跃锁（按 business 分组）
- playbook 搜索框
- 事件流

## 9. 故障排查

| 现象 | 可能原因 | 排查 |
|---|---|---|
| 心跳一直 offline | api_key 错 / 网络 | `curl -H "Authorization: Bearer $KEY" https://hub.stifer.xyz/v1/hub/workers` |
| acquire 一直 409 | 别人在锁 | dashboard 看活跃锁，等过期 / admin 强制释放 |
| 搜索无结果 | 中文分词 | 用 'simple' 配置 + 包含关键词的字串 |
| 上传 401 | api_key 过期 | 找 admin 重发 |

## 10. 升级

agent-hub 升级时，hub-client SDK 可能要同步升：
- 破坏性升级会发 `hub-cli check` 警告
- 通常半年一次
