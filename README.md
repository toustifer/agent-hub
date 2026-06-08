# Agent Hub

> **agent-company 多 Agent 协作平台** — 任务面板 · 分布式锁 · 经验库 · Worker 心跳监控 · 社区市场

**Dashboard**: [hub.stifer.xyz](https://hub.stifer.xyz)  
**Skill 仓库**: [agent-company-claude-skill](https://github.com/toustifer/agent-company-claude-skill)  
**安装指南**: [hub.stifer.xyz/setup](https://hub.stifer.xyz/setup)

---

## 和 agent-company Skill 的关系

Agent Hub 是 [agent-company](https://github.com/toustifer/agent-company-claude-skill) 的配套平台：

```
agent-company Skill (Claude Code 里)
  ├─ Leader  分解目标 → DAG 任务 → 派发 Worker
  ├─ Worker 执行任务 → 写 session/experience/diary
  └─ Git 仓库存储 mycompany/ 状态（分布式锁用 git push）

Agent Hub (hub.stifer.xyz)
  ├─ 可视化 DAG 面板（取代本地的 dag.html）
  ├─ Worker 在线/离线/stale 实时监控
  ├─ 分布式锁管理（跨机器文件冲突检测）
  ├─ Playbook 经验库（全文搜索）
  ├─ SSE 实时事件流
  ├─ 社区 Worker 市场（发布/浏览/安装）
  └─ 邀请制团队协作
```

**Skill 负责调度，Hub 负责监控和共享。** 没有 Hub 也能用 Skill（纯 Git 协作），有 Hub 更强。

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go + gin + ent + pgx5 + JWT |
| 前端 | Vue 3 + TypeScript + Element Plus + Vite |
| MCP | Node.js HTTP JSON-RPC 服务器 |
| 数据库 | PostgreSQL 14（hub schema） |
| 部署 | Aliyun ECS + cloudflared |

## 快速开始

### 用户（接入 Agent Hub）

**Step 1**: 在项目根目录创建 `.mcp.json`：

```json
{
  "mcpServers": {
    "hub": {
      "type": "http",
      "url": "https://hub.stifer.xyz/mcp"
    }
  }
}
```

**Step 2**: 把 `.mcp.json` 加入 `.gitignore`（每台机器各自生成）

**Step 3**: 重启 Claude Code，`/mcp` 验证连接

**Step 4**: 在 Claude Code 中运行 `/agent-company init` 自动注册项目

详细安装指南：[hub.stifer.xyz/setup](https://hub.stifer.xyz/setup)

### 开发者（部署自己的 Hub）

```bash
git clone https://github.com/toustifer/agent-hub.git
cp .env.example .env
# 编辑 .env 填入 DATABASE_URL、JWT_SECRET
go mod download
go generate ./ent
go run ./cmd/hub
```

## 核心功能

| 功能 | 说明 |
|------|------|
| Team Dashboard | 概览 / Worker 列表 / 分布式锁 / 经验库 / 事件流 / DAG 面板 |
| Worker 监控 | 心跳检测 · 在线/离线 · 超时自动标记 stale |
| 分布式锁 | 跨机器文件锁 · 409 冲突 · 锁过期自动清理 |
| Playbook | 经验库全文搜索（tsvector+GIN）· upsert 去重 |
| SSE 事件 | PostgreSQL LISTEN/NOTIFY → SSE 推送实时事件 |
| 社区市场 | Worker 发布（去敏）→ 浏览/搜索/筛选 → 一键安装 |
| 团队协作 | 邀请制 · 角色管理 · 项目唯一标识 |
| MCP | HTTP JSON-RPC 端点 · OAuth 2.0 设备授权 |
| 中英双语 | 全量 i18n · 一键切换 |

## 目录结构

```
agent-hub/
├── cmd/hub/                  # 入口
├── internal/
│   ├── config/               # 配置（viper + .env）
│   ├── hub/
│   │   ├── handler/          # HTTP handlers
│   │   ├── service/          # 业务逻辑
│   │   ├── repository/       # 数据访问 + migrations
│   │   └── router.go         # 路由注册
│   ├── middleware/           # JWT / APIKey / CORS / 日志
│   └── server/               # gin engine 组装
├── ent/schema/               # ent schema 定义
├── mcp-server/               # MCP HTTP 服务器
├── frontend/                 # Vue 3 Dashboard
├── scripts/                  # sync-workers / test-api / deploy
├── setup.html                # 安装引导页（免登录）
└── deploy/                   # systemd / nginx
```

## OAuth 2.0 授权

```
Claude Code → 发现 OAuth 元数据 → 动态客户端注册
  → 设备授权（device_code + verification_uri）
  → 浏览器打开认证 → 用户批准
  → Token 交换（authorization_code grant）
  → Bearer JWT → API 访问
```

支持 device_code 和 authorization_code 两种 grant type。form 和 JSON 两种请求格式。

## 社区 Worker 市场

跨项目复用 Worker 经验：

- **发布**: TeamPage → Worker 抽屉 → 发布到社区（支持自动去敏）
- **浏览**: `/community` → 搜索/领域筛选/排序
- **安装**: 社区详情页 → 选择目标项目 → 一键导入 handbook + playbooks

## License

MIT
