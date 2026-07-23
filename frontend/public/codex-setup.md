# Agent Hub on Codex CLI

> Paste this whole file into Codex.  
> Base: https://hub.stifer.xyz  
> Updated: 2026-07-23  
> Verified on Codex CLI: `codex mcp add hub --url …` + OAuth (`codex mcp login hub`) succeeds against production.

## What you get

| Capability | Codex |
|------------|-------|
| Hub remote MCP (docs / events / branches / templates / invites) | **Yes** |
| OAuth login to Hub | **Yes** (`codex mcp login hub`) |
| Bearer JWT via env | **Yes** (`--bearer-token-env-var`) |
| soft-sync `~/.agent-hub/config.json` | **Yes** (host-agnostic; needs agentflow for lifecycle writes) |
| agentflow stdio tools | **Optional** (section B) |
| Claude sticky `/agentflow on` | **No** — Claude-only flagship |

Team `business_code` is the **auto-generated 4-char short code** on the team card (not the display name, not `/team/name-code`).

---

## A. Team collab only (recommended first)

### 1) Add Streamable HTTP MCP

```bash
codex mcp add hub --url https://hub.stifer.xyz/mcp
```

Codex may auto-start OAuth. If it does not:

```bash
codex mcp login hub
```

Approve in the browser. Production Hub advertises OAuth via:

- `GET https://hub.stifer.xyz/.well-known/oauth-authorization-server`
- DCR: `POST https://hub.stifer.xyz/v1/hub/oauth/register`
- Authorize + token endpoints in metadata  
- Resource: `https://hub.stifer.xyz/mcp`

### 2) Fallback — Bearer JWT in env

If OAuth is unavailable on a machine:

1. Log in at https://hub.stifer.xyz or complete device login (`POST /v1/hub/auth/device` → confirm → token).  
2. Export the JWT (do **not** commit it):

**Windows (PowerShell):**

```powershell
$env:HUB_TOKEN = "<jwt>"
# permanent for new shells:
setx HUB_TOKEN "<jwt>"
```

**macOS / Linux:**

```bash
export HUB_TOKEN="<jwt>"
```

3. Re-add with env binding:

```bash
codex mcp remove hub
codex mcp add hub --url https://hub.stifer.xyz/mcp --bearer-token-env-var HUB_TOKEN
```

### 3) Config shape (what Codex writes)

After a successful add, `~/.codex/config.toml` contains:

```toml
[mcp_servers.hub]
url = "https://hub.stifer.xyz/mcp"
```

OAuth tokens are stored by Codex (not in this TOML).  
With bearer fallback:

```toml
[mcp_servers.hub]
url = "https://hub.stifer.xyz/mcp"
bearer_token_env_var = "HUB_TOKEN"
```

Inspect:

```bash
codex mcp get hub --json
codex mcp list
```

Expect something like:

```text
Name  Url                         Status   Auth
hub   https://hub.stifer.xyz/mcp  enabled  OAuth
```

(`codex mcp get` transport type is `streamable_http`.)

### 4) Verify in a session

1. Start Codex in any project.  
2. Call `hub_list_my_businesses` → teams + **4-char** codes.  
3. Call `hub_list_docs` / `hub_list_events` with `business_code` from step 2.  
4. Optional: `hub_export_soft_sync_config` with that code → write `~/.agent-hub/config.json` for agentflow soft-sync.

### 5) Useful tools (human JWT)

| Tool | Notes |
|------|--------|
| `hub_list_my_businesses` | Discover codes |
| `hub_list_docs` / `hub_get_doc` / `hub_upsert_doc` | Team docs |
| `hub_list_events` / `hub_append_event` | Work log |
| `hub_list_branches` / `hub_report_branches` | Branch index |
| `hub_list_worker_templates` / `hub_publish_worker_template` | Templates |
| `hub_invite_member` | Admin |
| `hub_export_soft_sync_config` | JWT only (not API key) |

---

## B. Optional — agentflow stdio + soft-sync

Hub collab does **not** require agentflow. Add this only if you want the local task engine and automatic soft-sync to the board.

### B.1 Build binary (no Claude skill required)

```bash
git clone https://github.com/toustifer/agentflow.git
cd agentflow
```

**Windows:**

```powershell
$bin = "$env:USERPROFILE\.agentflow\bin"
New-Item -ItemType Directory -Force -Path $bin | Out-Null
go build -o "$bin\agentflow.exe" .\cmd\agentflow\
```

**macOS / Linux:**

```bash
mkdir -p ~/.agentflow/bin
go build -o ~/.agentflow/bin/agentflow ./cmd/agentflow/
```

### B.2 Register stdio MCP in Codex

**Windows:**

```bash
codex mcp add agentflow -- "%USERPROFILE%\.agentflow\bin\agentflow.exe" stdio
```

(If your shell does not expand `%USERPROFILE%`, pass the absolute path.)

**macOS / Linux:**

```bash
codex mcp add agentflow -- $HOME/.agentflow/bin/agentflow stdio
```

### B.3 soft-sync config (host-agnostic)

```json
{
  "hub_url": "https://hub.stifer.xyz",
  "token": "<Hub JWT after OAuth or device login>",
  "business_code": "<4-char team code>",
  "email": "<your@email>"
}
```

Write to `~/.agent-hub/config.json` (or export via `hub_export_soft_sync_config`).  
Old hand-filled codes work ~90 days via alias after the short-code migration.

Disable: `HUB_SYNC=0` / `HUB_DISABLED=1`.

### B.4 Orchestration without Claude sticky

- Codex **skills**: put rules under `~/.codex/skills/agentflow/SKILL.md` if you want on-demand playbooks.  
- Project **`AGENTS.md`**: remind the model to use Hub tools and 4-char codes.  
- **Do not** expect Claude’s `/agentflow on` UserPromptSubmit hooks to work 1:1.

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `Auth Unsupported` / tools unauthorized | `codex mcp login hub` or re-add with `--bearer-token-env-var HUB_TOKEN` |
| OAuth browser fails | Confirm Hub is up; try resource explicitly: `codex mcp add hub --url https://hub.stifer.xyz/mcp --oauth-resource https://hub.stifer.xyz/mcp` then `codex mcp login hub` |
| Wrong team | Use 4-char code from Dashboard / `hub_list_my_businesses`, not display name |
| soft-sync silent | Check `~/.agent-hub/config.json` token + code; ensure agentflow is the process doing task transitions |
| Expect sticky mode | Use Claude Code for sticky; Codex uses skill/AGENTS only |

---

## Do not

- Treat MCP-only as a full Claude agentflow install  
- Put display name or `/team/…` path into `business_code`  
- Use the legacy Node `mcp-server` stdio bridge for new setups  
- Commit JWTs or API keys  

---

## Related

- Master AI paste (Claude-first): https://hub.stifer.xyz/agent-setup.md  
- agentflow local engine: https://hub.stifer.xyz/agentflow-setup.md  
- MCP protocol notes: https://hub.stifer.xyz/mcp.md  
- Portal docs UI: https://hub.stifer.xyz/app (Official Docs)
