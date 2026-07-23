# Agent Hub MCP — configuration guide for AI agents

> **Audience:** coding agents (Claude Code, Cursor, etc.) and humans pasting into AI context.  
> **Canonical URL:** https://hub.stifer.xyz/mcp.md  
> **Hub base URL:** https://hub.stifer.xyz  
> **MCP endpoint (primary):** `https://hub.stifer.xyz/mcp` — official Streamable HTTP in the Go Hub process  
> **Last product model:** remote MCP + OAuth/JWT; local Node stdio bridge is **legacy**.

If you were told to “connect this project to Agent Hub”, follow this file top to bottom. Do not invent API keys.

---

## 0. Goal checklist

After success you should have:

1. MCP server `hub` listed as connected (`/mcp` in Claude Code) pointing at **remote URL**.
2. Host completed OAuth (preferred) **or** you use `Authorization: Bearer <jwt>` after `hub_login` / device flow.
3. Optional: `business_code` of the user’s team (from Dashboard or `hub_list_my_businesses`).
4. Ability to call human tools (invite, list/report branches, dag) **without** a local `node mcp-server`.

---

## 1. Primary path — remote Streamable HTTP (no local Node)

Claude / Cursor connect **directly** to Hub. No clone, no `npm install`, no stdio bridge.

### 1.1 Claude Code project `.mcp.json`

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

Host will discover OAuth via:

- `GET https://hub.stifer.xyz/.well-known/oauth-authorization-server`
- DCR: `POST https://hub.stifer.xyz/v1/hub/oauth/register`
- Device / authorize + token endpoints listed in metadata

After OAuth, the Host sends `Authorization: Bearer <access_token>` on MCP requests. Access tokens are Hub JWTs (same as REST).

### 1.2 Manual Bearer (Inspector / curl / debugging)

1. Register / log in at https://hub.stifer.xyz  
2. Device login: `POST /v1/hub/auth/device` → open `verification_url` → confirm → `GET /v1/hub/auth/device/token?code=...`  
   Or call tool `hub_login` (step 1 URL, step 2 exchange code).  
3. Pass header: `Authorization: Bearer <jwt>`

Smoke:

```bash
# initialize
curl -sS -X POST https://hub.stifer.xyz/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
# tools/list (include Mcp-Session-Id from initialize response if required)
```

### 1.3 Restart / reconnect

1. Restart Claude Code (or reconnect MCP).  
2. Run `/mcp` → expect `hub` connected to `https://hub.stifer.xyz/mcp`.  
3. If disconnected: check OAuth approval, network, and that production Hub is running the Go binary (not only the old Node `:9001` process).

---

## 2. Authenticate

### 2.1 OAuth (preferred for Claude remote)

Host drives DCR + authorization_code or device_code. User approves in browser at `/auth/device`. Token endpoint returns `access_token` (Hub JWT, ~72h).

### 2.2 Device login via tool `hub_login`

```text
Step A: hub_login() → verification URL + code
Step B: open URL; log in; Approve
Step C: hub_login({ "code": "<code>" }) → JWT text (also usable as Bearer)
```

### 2.3 Machine tools (API key)

Only for: `hub_heartbeat`, `hub_acquire/release/renew_lock`, `hub_append_event`, `hub_create_playbook`.

Headers:

```http
X-API-Key: <key>
X-Business-Code: <code>
```

Human JWT path remains primary for Dashboard / Claude.

---

## 3. Tool matrix (24 tools, parity with prior Node server)

| Tool | Auth |
|------|------|
| `hub_login` | none |
| `hub_list_my_businesses` | JWT |
| `hub_heartbeat` | API key or JWT membership |
| `hub_acquire_lock` / `hub_release_lock` / `hub_renew_lock` | API key or JWT |
| `hub_append_event` / `hub_create_playbook` | API key or JWT membership |
| `hub_search_playbooks` | JWT or API key |
| `hub_list_workers` / `hub_list_locks` / `hub_list_events` | JWT membership |
| `hub_add_repo` / `hub_sync_dag` / `hub_get_dag` | JWT membership |
| `hub_invite_member` | JWT admin |
| `hub_accept_invitation` | JWT |
| `hub_create_link_request` | JWT |
| `hub_list_link_requests` / `hub_review_link_request` | JWT admin |
| `hub_report_branches` / `hub_list_branches` / `hub_bind_branch` / `hub_refresh_branches` | JWT membership |

Membership is enforced server-side (`hub_memberships`). Non-members get tool errors.

---

## 4. Legacy — local stdio Node bridge

Still works for offline/dev:

```json
{
  "mcpServers": {
    "hub": {
      "command": "node",
      "args": ["ABS_PATH/mcp-server/index.js"],
      "env": { "HUB_API_URL": "https://hub.stifer.xyz" }
    }
  }
}
```

- `mcp-server/index.js` — stdio MCP (legacy).  
- `mcp-server/http-server.js` on `:9001` — **deprecated**; Hub no longer reverse-proxies `/mcp` to it.

Prefer remote URL for all new setups.

---

## 5. agentflow boundary

- **agentflow** = local task engine (SQLite / worktree). Soft-sync to Hub uses **REST + JWT** only.  
- **Hub remote MCP** = collaboration tools for Claude (teams, locks, branches, dag).  
- Do **not** require Node Hub MCP for agentflow soft-sync.

---

## 6. Troubleshooting

| Symptom | Fix |
|---------|-----|
| 502 / connection refused on `/mcp` | Deploy Go Hub with in-process MCP; stop relying on Node `:9001` |
| `tools/list` empty after OAuth | Confirm token is Hub JWT (`uid` claim); re-login |
| 403 / not a member | Join/invite business; check `hub_list_my_businesses` |
| OAuth metadata mismatch | Issuer/endpoints from `HUB_PUBLIC_URL` (default `https://hub.stifer.xyz`) |

---

## 7. Related docs

- [MCP_OFFICIAL.md](./MCP_OFFICIAL.md) — product decision + architecture  
- https://modelcontextprotocol.io/docs/learn/architecture  
- Public mirror: https://hub.stifer.xyz/mcp.md  
