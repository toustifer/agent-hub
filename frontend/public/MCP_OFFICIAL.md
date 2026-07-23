# MCP 官方形态（经验沉淀）

> 对照 [MCP Intro](https://modelcontextprotocol.io/docs/getting-started/intro) 与 [Architecture](https://modelcontextprotocol.io/docs/learn/architecture)。  
> **产品决策（2026-07）：** 人侧 / Claude 接 Hub **按官方 remote MCP**，不再以「本机 stdio 桥封装 REST」为主路径。

## 1. 官方事实（勿再讲错）

| 事实 | 说明 |
|------|------|
| MCP 有状态 | initialize → 能力协商 → 会话内连接与 capabilities 由 Host 维护 |
| Server 可本地可远程 | 本地常见 **stdio**；远程用 **Streamable HTTP** |
| 远程鉴权 | Bearer / API Key / 自定义头；**推荐 OAuth 取 token** |
| Host 职责 | 每个 MCP server 一个 client 连接；会话期内记住连接 |
| 协议不管业务 | MCP 不规定 token 必须落盘 `~/.agent-hub`；那是我们以前的实现细节 |

**禁止再向用户解释成：**「Claude 记不住登录所以必须本机桥」。那是错误说法。

## 2. 已实现状态（2026-07-22）

| 项 | 状态 |
|----|------|
| Go 进程内 Streamable HTTP `@ /mcp` | **已上线代码路径** — `internal/mcp` + `github.com/modelcontextprotocol/go-sdk` |
| 去掉 `:9001` reverse proxy | **已做** — `internal/hub/router.go` 挂载 `gin.WrapH(mcpHub.HTTPHandler())` |
| 工具面 | **24 工具** 对齐原 Node `tools.js`（进程内 service/SQL） |
| OAuth metadata | issuer/endpoints 来自 `HUB_PUBLIC_URL`；grant_types 含 `authorization_code` + device_code |
| DCR | `POST /v1/hub/oauth/register` 持久化 `hub.hub_oauth_clients`（migration 0010） |
| Device / token | 既有 `hub_device_codes` + `OAuthDeviceAuthorize` / `OAuthDeviceToken` |
| 文档 / UI 主路径 | remote URL（见 `MCP_FOR_AI.md`、`McpSetup.vue`） |
| Node stdio / http-server.js | **legacy**，生产不再依赖 `:9001` |

```text
当前拓扑:

  Claude / Codex / Cursor ──Streamable HTTP + OAuth/Bearer──►  https://hub.stifer.xyz/mcp
                                                                    │
                                                              (同进程 Go Hub)
                                                                    │
                                                              internal/mcp tools
                                                              → service / SQL
                                                              JWT + membership

  agentflow 本地引擎 ── REST soft-sync ──► hub.stifer.xyz /v1/hub/*
```

**Host 配置入口**

| Host | 配置 |
|------|------|
| Claude Code | `.mcp.json` `type: http` + URL（见 [MCP_FOR_AI.md](./MCP_FOR_AI.md)） |
| **Codex CLI** | `codex mcp add hub --url https://hub.stifer.xyz/mcp` + `codex mcp login hub` → [codex-setup.md](../frontend/public/codex-setup.md) / https://hub.stifer.xyz/codex-setup.md |
| Cursor | 同 Claude remote HTTP MCP |

## 3. 与 agentflow 的边界

- **agentflow** = 本机执行真相（SQLite / worktree）。
- **Hub remote MCP** = 协作索引与 Dashboard 的 **官方工具面**。
- 软投影（task/branch）可以是 agentflow → REST，不强制 Claude 再经本地桥。
- 若 Claude 只需团队/DAG/分支协作：只配 **remote hub MCP** 即可。
- 若 Claude 要跑本地任务引擎：仍配 **agentflow stdio**（那是本地引擎，不是 Hub 的“多余桥”）。

## 4. 参考链接

- https://modelcontextprotocol.io/docs/getting-started/intro  
- https://modelcontextprotocol.io/docs/learn/architecture  
- Spec / SDK：https://modelcontextprotocol.io/llms.txt  
- 配置指南：[MCP_FOR_AI.md](./MCP_FOR_AI.md)

## 5. 迁移原则

1. 新用户默认路径 = remote MCP，零本地 `mcp-server` 安装。  
2. 旧 stdio 桥可短期兼容，文档降级为 “legacy”。  
3. 鉴权与 membership 仍以 Hub 服务端为准（JWT + membership），不因 MCP 形态放松。  
4. **不要**再把「假 HTTP JSON-RPC 且无 initialize 生命周期」的 `http-server.js` 当作正式方案。
