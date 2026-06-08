# agent-hub Client SDK

多语言客户端 SDK，供业务项目集成 agent-hub 平台。

## 安装

SDK 为零依赖的独立文件，可直接复制到项目中：

```bash
# Node.js
cp sdk/hub-client.js your-project/lib/

# Python
cp sdk/hub_client.py your-project/
```

## Node.js

```javascript
const { HubClient } = require('./hub-client');

const hub = new HubClient({
  baseUrl: 'https://hub.stifer.xyz',
  apiKey: 'your-api-key',
  businessCode: 'siruoning',
});

// Worker 心跳
await hub.heartbeat({ worker_id: 'my-worker', version: '1.0.0', host: 'dev', pid: 1234 });

// 分布式锁
const lock = await hub.acquireLock({ resource_key: 'mylock', worker_id: 'my-worker', ttl_seconds: 300 });
await hub.renewLock({ holder_token: lock.holder_token, ttl_seconds: 300 });
await hub.releaseLock({ holder_token: lock.holder_token });

// Playbook 知识库（创建 / 搜索 / 查询）
await hub.createPlaybook({ category: 'patterns', title: 'My Pattern', content: '# Content', tags: ['api'] });
const results = await hub.searchPlaybooks({ q: 'pattern', limit: 10 });
const detail = await hub.getPlaybook(1);

// 事件
await hub.appendEvent({ event_type: 'task_completed', payload: { task: 'T-1' } });

// 业务查询
const biz = await hub.getBusinessByCode('siruoning');
```

### 认证方式

| 场景 | 构造参数 | 发送的请求头 |
|------|----------|-------------|
| Worker 端 | `apiKey` + `businessCode` | `X-API-Key` + `X-Business-Code` |
| Admin 端 | `token` (JWT) | `Authorization: Bearer <token>` |
| 两者同时 | 全部设置 | 三种头同时发送 |

### 错误处理

```javascript
try {
  await hub.acquireLock({ resource_key: 'critical', worker_id: 'w1' });
} catch (err) {
  if (err.code === 409) {
    console.error('锁冲突:', err.message);
  } else {
    console.error('请求失败:', err.code, err.message);
  }
}
```

## Python

```python
from hub_client import HubClient, HubLockError

hub = HubClient(
    base_url="https://hub.stifer.xyz",
    api_key="your-api-key",
    business_code="siruoning",
)

# Worker 心跳
hub.heartbeat(worker_id="my-worker", version="1.0.0", host="dev", pid=1234)

# 分布式锁
lock = hub.acquire_lock(resource_key="mylock", worker_id="my-worker", ttl_seconds=300)
hub.renew_lock(holder_token=lock["holder_token"], ttl_seconds=300)
hub.release_lock(holder_token=lock["holder_token"])

# Playbook 知识库（创建 / 搜索 / 查询）
hub.create_playbook(category="patterns", title="My Pattern", content="# Content", tags=["api"])
results = hub.search_playbooks(q="pattern", limit=10)
detail = hub.get_playbook(1)

# 事件
hub.append_event(event_type="task_completed", payload={"task": "T-1"})

# 业务查询
biz = hub.get_business_by_code("siruoning")
```

### 认证方式

| 场景 | 构造参数 | 发送的请求头 |
|------|----------|-------------|
| Worker 端 | `api_key` + `business_code` | `X-API-Key` + `X-Business-Code` |
| Admin 端 | `token` (JWT) | `Authorization: Bearer <token>` |
| 两者同时 | 全部设置 | 三种头同时发送 |

### 错误处理

```python
try:
    hub.acquire_lock(resource_key="critical", worker_id="w1")
except HubLockError as e:
    print(f"锁冲突: {e}, 当前持有者: {e.holder_worker_id}")
except Exception as e:
    print(f"请求失败: {e}")
```

## API 参考

所有 Worker 端方法均自动注入 `business_code`（从构造函数获取）。

### Worker 心跳

| 语言 | 方法 |
|------|------|
| Node.js | `hub.heartbeat({ worker_id, version, host, pid })` |
| Python | `hub.heartbeat(worker_id, version, host, pid)` |

### 分布式锁

| 操作 | Node.js | Python |
|------|---------|--------|
| 获取锁 | `hub.acquireLock({ resource_key, worker_id, ttl_seconds? })` | `hub.acquire_lock(resource_key, worker_id, ttl_seconds=300)` |
| 续期锁 | `hub.renewLock({ holder_token, ttl_seconds? })` | `hub.renew_lock(holder_token, ttl_seconds=300)` |
| 释放锁 | `hub.releaseLock({ holder_token })` | `hub.release_lock(holder_token)` |

锁冲突时 Node.js 抛出 `err.code === 409` 的 Error；Python 抛出 `HubLockError` 异常（含 `resource_key`、`holder_worker_id` 属性）。

### Playbook 知识库

| 操作 | Node.js | Python |
|------|---------|--------|
| 创建 | `hub.createPlaybook({ category, title, content, tags?, worker_id? })` | `hub.create_playbook(category, title, content, tags=None, worker_id=None)` |
| 搜索 | `hub.searchPlaybooks({ q?, category?, limit?, offset? })` | `hub.search_playbooks(q=None, category=None, limit=20, offset=0)` |
| 单条 | `hub.getPlaybook(id)` | `hub.get_playbook(id_)` |

### 事件

| 操作 | Node.js | Python |
|------|---------|--------|
| 追加 | `hub.appendEvent({ event_type, actor?, payload? })` | `hub.append_event(event_type, actor=None, payload=None)` |

### Business

| 操作 | Node.js | Python |
|------|---------|--------|
| 按 code 查询 | `hub.getBusinessByCode(code)` | `hub.get_business_by_code(code)` |

## 依赖

- **Node.js**：零外部依赖，仅使用 `http` / `https` / `crypto` / `url` 内置模块。
- **Python**：零外部依赖，仅使用 `urllib.request` / `json` / `uuid` 标准库，兼容 Python 3.8+。
