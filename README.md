# agent-hub

> 多业务代码协作协调层 — 让多个 AI 管理的业务域在同一台服务器上互不打架

## 它是什么

一个独立的 Go 服务，作为 sub2api v0.1.104 之上的**业务协调模块**存在。

提供 4 个核心能力：

1. **业务注册表** — 哪些业务（siruoning / insight-tutor / portal）接入 hub
2. **Worker 心跳** — 哪个 AI 正在跑、跑什么任务、什么时候离线
3. **分布式锁** — 跨业务 / 跨 Worker 抢占同一资源（文件 / 业务域）时互斥
4. **Playbook 知识库** — 跨业务沉淀架构决策、踩坑、模式

## 它不是

- 不是 sub2api 的替代品
- 不是 Claude Code 的替代品
- 不是 git 的替代品
- 不是产品 CLI（不做 Telegram / Discord 网关）

## 架构

```
┌────────────────────────────────────────────────────────┐
│  47.115.134.24 (Aliyun ECS, 1.6GB RAM)                 │
│                                                         │
│  ┌──────────────┐  ┌────────────────┐  ┌────────────┐  │
│  │ sub2api      │  │ ★ agent-hub    │  │ insight-   │  │
│  │ v0.1.104     │  │ 端口 9000       │  │ tutor      │  │
│  │ 端口 8080    │  │ ~30MB           │  │ 端口 8000  │  │
│  │ ~50MB        │  │                 │  │ Docker     │  │
│  └──────┬───────┘  └────────┬────────┘  └─────┬──────┘  │
│         │                   │                  │         │
│         └─────────────┬─────┘                  │         │
│                       │                        │         │
│         ┌─────────────▼──────────────┐         │         │
│         │  PostgreSQL 14 (宿主机)     │◄────────┘         │
│         │  schemas:                   │                   │
│         │   public  → sub2api 的表    │                   │
│         │   public  → insight-tutor 表│                   │
│         │   ★ hub  → agent-hub 的表   │                   │
│         └────────────────────────────┘                   │
│         ┌────────────────────────────┐                   │
│         │  Redis 6379 (宿主机)        │                   │
│         └────────────────────────────┘                   │
│                          │                               │
│                  ┌───────▼────────┐                      │
│                  │  cloudflared    │                     │
│                  │  api.stifer.xyz │                     │
│                  │  sub2api.stifer │                     │
│                  │  ★ hub.stifer   │                     │
│                  │  insight.stifer │                     │
│                  └─────────────────┘                     │
└────────────────────────────────────────────────────────┘
```

## 核心约束

| 约束 | 说明 |
|---|---|
| sub2api 0 修改 | 通过 `go.mod replace` 复用 sub2api 包，不动其源码 |
| 共享数据库 | 同 PostgreSQL，用 `hub` schema 隔离 |
| 共享 Redis | 锁 + 限流复用 sub2api 的 redis 实例 |
| 共享认证 | JWT / APIKey 调 sub2api 的 service 校验 |

## 目录

```
agent-hub/
├── cmd/hub/main.go              # 入口
├── internal/
│   ├── config/                  # 配置加载
│   ├── hub/
│   │   ├── schema/              # 5 个 ent schema
│   │   ├── handler/             # gin handler
│   │   ├── service/             # 业务逻辑
│   │   ├── repository/          # 数据访问
│   │   └── router.go            # 路由注册
│   ├── middleware/              # JWT / APIKey 复用 sub2api
│   └── server/                  # gin engine 组装
├── ent/                         # ent 生成代码（不手写）
├── migrations/                  # SQL migration（partial index / tsvector）
├── frontend/                    # Vue 3 Hub Dashboard
├── scripts/                     # 部署 / 启动脚本
├── deploy/                      # systemd unit / nginx config
├── docs/                        # 架构 / 接入文档
└── go.mod                       # require + replace sub2api
```

## 快速开始（开发者）

```bash
# 1. 拉 sub2api 源码到本地（用于 go.mod replace）
git clone https://github.com/Wei-Shaw/sub2api.git /opt/sub2api-src

# 2. 本地开发
cp .env.example .env
go mod download
go generate ./ent
go run ./cmd/hub

# 3. 构建
go build -o bin/hub ./cmd/hub
```

## 快速开始（业务接入）

```bash
# 1. 注册业务
curl -X POST https://hub.stifer.xyz/v1/hub/businesses \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"code":"siruoning","name":"AI 智能药盒","repo_url":"git@github.com:..."}'

# 2. 创建 APIKey
curl -X POST https://hub.stifer.xyz/v1/hub/apikeys \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"business_code":"siruoning","name":"worker-medication"}'

# 3. Worker 启动时接入（业务仓内）
node .mycompany/hub-boot.js
```

## 路线图

| 阶段 | 内容 | 工期 |
|---|---|---|
| Phase 0 | 项目骨架 | 0.5 天 |
| Phase 1 | 5 个 ent schema + 索引 | 2 天 |
| Phase 2 | 6 个 service（business/worker/lock/playbook/event） | 2 天 |
| Phase 3 | HTTP 层（handler + middleware + router） | 1 天 |
| Phase 4 | Hub Dashboard 前端 | 1-2 天 |
| Phase 5 | hub-client SDK（js + py） | 1-2 天 |
| Phase 6 | 部署（systemd + nginx + cloudflared） | 1 天 |
| Phase 7 | 跨业务接通 + 文档 | 1 周 |

## 详细文档

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — 架构决策、设计原则、与 sub2api 的边界
- [docs/PLAN.md](docs/PLAN.md) — 完整实施计划（含决策历史）
- [docs/INTEGRATION.md](docs/INTEGRATION.md) — 业务接入指南（worker 端）
- [docs/OPERATIONS.md](docs/OPERATIONS.md) — 运维手册（备份、升级、故障恢复）

## 状态

🚧 Phase 0 进行中
