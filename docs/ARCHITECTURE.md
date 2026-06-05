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
-- 初始化
CREATE SCHEMA IF NOT EXISTS hub;

-- 表引用
hub.hub_businesses
hub.hub_workers
hub.hub_locks
hub.hub_playbooks
hub.hub_events
```

跨 schema 引用 user / api_key：
```sql
-- hub 表只存 user_id int64，不建外键
-- 校验时调 sub2api service 实时验证
SELECT email, role, status FROM public.users WHERE id = $1
```

### P3：共享 Redis
sub2api 已经在用宿主机 redis（127.0.0.1:6379），agent-hub 复用同一实例：
- 锁 TTL 倒计时
- 限流
- 事件 SSE 推送队列

### P4：认证复用
不重新实现 JWT / APIKey，直接调 sub2api 的 service：

```go
// 伪代码
import "github.com/Wei-Shaw/sub2api/internal/service"

authService := sub2apiService.NewAuthService(cfg, db, redis)
subject, err := authService.ValidateToken(ctx, token)
```

耦合面控制在 3-4 个最稳定方法：
- `AuthService.ValidateToken(token) → (Subject, error)`
- `APIKeyService.ValidateKey(key) → (User, error)`
- `UserService.GetUser(id) → (User, error)`

## 5 个核心表

### hub_businesses — 业务注册表
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| code | varchar(64) unique | 业务代号（siruoning） |
| name | varchar(128) | 业务名 |
| repo_url | text | 仓库 URL |
| owner_user_id | bigint | 关联 sub2api user.id |
| description | text | 业务说明 |
| status | varchar(20) | active / suspended |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### hub_workers — Worker 心跳
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| business_id | bigint FK | 关联 business |
| worker_id | varchar(128) | 业务内的 worker 名（medication） |
| version | varchar(32) | agent-company 版本 |
| last_heartbeat_at | timestamptz | 最后心跳 |
| status | varchar(20) | online / offline / dead |
| created_at | timestamptz | |
| updated_at | timestamptz | |

unique: (business_id, worker_id)

### hub_locks — 分布式锁
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| business_id | bigint FK | |
| resource_key | varchar(256) | 锁的资源（如 medication.pages.Homepage） |
| holder_token | varchar(64) | 持有者 token（每次 acquire 重新生成） |
| holder_worker_id | bigint | 持有者 worker |
| acquired_at | timestamptz | |
| expires_at | timestamptz | 锁过期时间 |
| heartbeat_at | timestamptz | 最后续期 |
| released_at | timestamptz | 主动释放（NULL = 还锁着） |

**partial unique index**：
```sql
CREATE UNIQUE INDEX hub_locks_active_resource
  ON hub.hub_locks (resource_key)
  WHERE released_at IS NULL AND expires_at > now();
```

### hub_playbooks — 知识库
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| business_id | bigint FK | NULL 表示跨业务 |
| category | varchar(32) | decisions / patterns / gotchas |
| title | varchar(256) | |
| content | text | Markdown |
| tags | text[] | |
| tsv | tsvector | 全文搜索 |
| created_by_worker_id | bigint | 上传的 worker |
| created_at | timestamptz | |
| updated_at | timestamptz | |

GIN index on tsv

### hub_events — 审计日志
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigserial | 主键 |
| business_id | bigint FK | |
| actor | varchar(128) | 谁触发（worker / user） |
| event_type | varchar(64) | lock.acquired / lock.released / heartbeat.missed ... |
| payload | jsonb | 任意数据 |
| created_at | timestamptz | |

append-only，不删除。

## 锁算法

### Acquire
```sql
-- 1. 清理过期锁（可异步，acquire 时也兜底）
UPDATE hub.hub_locks
SET released_at = now()
WHERE released_at IS NULL
  AND expires_at < now()
  AND resource_key = $1;

-- 2. 尝试插入
INSERT INTO hub.hub_locks (
  business_id, resource_key, holder_token, holder_worker_id,
  acquired_at, expires_at, heartbeat_at
)
SELECT b.id, $1, $2, $3, now(), now() + interval '$4 seconds', now()
FROM hub.hub_businesses b
WHERE b.code = $5
ON CONFLICT (resource_key) WHERE released_at IS NULL AND expires_at > now()
DO NOTHING
RETURNING id, expires_at;

-- 3. 如果没返回行 → 锁被别人持有
-- 4. 如果返回 → 自己拿到了，holder_token 已记录
```

### Renew
```sql
UPDATE hub.hub_locks
SET expires_at = now() + interval '$2 seconds',
    heartbeat_at = now()
WHERE holder_token = $1
  AND released_at IS NULL
  AND expires_at > now()
RETURNING id;
```

### Release
```sql
UPDATE hub.hub_locks
SET released_at = now()
WHERE holder_token = $1
  AND released_at IS NULL
RETURNING id;
```

## 路由

```
POST   /v1/hub/businesses                (admin JWT)
GET    /v1/hub/businesses                (admin JWT)
GET    /v1/hub/businesses/:code          (apikey)

POST   /v1/hub/apikeys                   (admin JWT)
GET    /v1/hub/apikeys                   (admin JWT)

POST   /v1/hub/workers/heartbeat         (apikey)
GET    /v1/hub/workers                   (admin JWT)
GET    /v1/hub/workers?business=...      (admin JWT)

POST   /v1/hub/locks/acquire             (apikey)
POST   /v1/hub/locks/renew               (apikey)
POST   /v1/hub/locks/release             (apikey)
GET    /v1/hub/locks                     (admin JWT)

POST   /v1/hub/playbooks                 (apikey)
GET    /v1/hub/playbooks/search?q=...    (apikey)
GET    /v1/hub/playbooks/:id             (apikey)

POST   /v1/hub/events                    (apikey)
GET    /v1/hub/events                    (admin JWT)
GET    /v1/hub/events/stream             (admin JWT, SSE)

GET    /health                           (public)
```

## 数据流

### Worker 启动
```
hub-boot.js
  → POST /v1/hub/workers/heartbeat (every 30s)
  → 返回 server_time, config_overrides
```

### Worker 执行关键操作前
```
withLock('medication.pages.Homepage', async () => {
  // POST /v1/hub/locks/acquire → holder_token
  // 跑业务
  // POST /v1/hub/locks/release
}, ttl=300)
```

### 冲突时
```
Worker A: POST /v1/hub/locks/acquire  → 200 {holder_token: "abc"}
Worker B: POST /v1/hub/locks/acquire  → 409 {holder: "Worker A", expires_at: ...}
Worker B: 等待 + 重试 / 放弃 / 看 hub_event
```

## 部署拓扑

```
                  ┌─────────────────┐
                  │  cloudflared    │
                  │  隧道 (TLS)     │
                  └────────┬────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   api.stifer.xyz    sub2api.stifer.xyz  hub.stifer.xyz
   (insight-tutor)   (sub2api 网关)      (★ agent-hub)
        │                  │                  │
        ▼                  ▼                  ▼
   :8000 (Docker)     :8080 (Go bin)     :9000 (Go bin)
```

## 后续阶段

- **阶段 2**：Playbook FTS5 中文搜索优化
- **阶段 3**：Honcho-style 跨 session 用户建模
- **阶段 4**：Cron 调度（定期跑 health check、数据巡检）
- **阶段 5**：多 Agent Dashboard 实时视图（WebSocket）
