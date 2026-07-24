# Agent setup for AI (Hub MCP + agentflow)

> Paste this whole file into Claude / Cursor.  
> **Codex CLI users:** prefer https://hub.stifer.xyz/codex-setup.md (same Hub, different host config).  
> Base: https://hub.stifer.xyz  
> Updated: 2026-07-24 · agentflow **v0.2.1** (download-first)

## Critical: agentflow install is a three-part bundle

**Personal local work requires ALL of:**

1. **Skill** — `/agentflow`, flows (`~/.claude/skills/agentflow/`)  
2. **MCP** — prebuilt stdio binary (`mcp__agentflow__*`)  
3. **Sticky mode hooks** — `/agentflow on` stays on for later turns (`mode-inject.js` in settings.json)

| Missing | Result |
|---------|--------|
| No skill | No `/agentflow` orchestration |
| No MCP | No tools |
| No hooks | `/agentflow on` does not stick; product incomplete |
| **All three** | Intended install |

Do **not** treat “only `.mcp.json` → agentflow.exe” as complete.  
Do **not** require users to `git clone` + `go build` (that is developer-only).

Full detail: https://hub.stifer.xyz/agentflow-setup.md

---

## Two scenarios

| Scenario | Need |
|----------|------|
| **1. Personal local work** | **Skill + MCP + sticky hooks** (must) |
| **2. Team collab only** | Hub remote MCP only |
| **1 + 2** | Bundle + Hub MCP; soft-sync optional via config file |

---

## Scenario 1 — full agentflow install (required for personal work)

**Default = download Release (no Go, no git clone).**

### macOS / Linux (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/toustifer/agentflow/master/scripts/install.sh \
  | VERSION=v0.2.1 bash -s -- --write-config
```

### Windows (PowerShell)

```powershell
$env:VERSION = 'v0.2.1'
irm https://raw.githubusercontent.com/toustifer/agentflow/master/scripts/install.ps1 | iex
# or: .\install.ps1 -WriteConfig
```

Installer:

1. Downloads `skill.tgz` + platform binary from  
   https://github.com/toustifer/agentflow/releases/tag/v0.2.1  
2. Installs to `~/.claude/skills/agentflow/` (+ `bin/agentflow`)  
3. Verifies **MCP GATE** is present  
4. Prints (or writes) `mcpServers.agentflow` with `args: ["stdio"]`

Then:

1. Merge sticky hooks printed by the script into `~/.claude/settings.json` (**do not** wipe other hooks)  
2. Fully quit and restart Claude Code  
3. Verify below  

### Sticky hooks (if installer only printed them)

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node /Users/YOU/.claude/skills/agentflow/hooks/mode-inject.js",
            "timeout": 5
          }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "node /Users/YOU/.claude/skills/agentflow/hooks/statusline.js",
    "refreshInterval": 5
  }
}
```

Windows: `node C:\\Users\\YOU\\.claude\\skills\\agentflow\\hooks\\...` and binary `...\\bin\\agentflow.exe`.

### Verify (all must pass — three layers)

| Layer | Check | Enough for work? |
|-------|--------|------------------|
| Config | `mcpServers.agentflow` in `~/.claude.json` | No |
| UI / process | `/mcp` lists agentflow and **not failed** | Required |
| **Session tools** | Model can call `mcp__agentflow__flow_ping` this turn | **Yes — only this** |

`claude mcp list` Connected, `agentflow:on` statusline, or Bash→stdio writing sqlite do **not** count.

1. Fully quit and restart Claude Code  
2. `/mcp` → agentflow **not failed**  
3. In-session: `mcp__agentflow__flow_ping` succeeds  
4. `grep -n "MCP GATE" ~/.claude/skills/agentflow/hooks/mode-lib.js` hits  
5. `/agentflow on` → sticky on; statusline may show `MCP:cfg|missing|broken`  
6. If MCP tools missing: **stop** — do **not** Bash/JSON-RPC/sqlite around the engine  

---

## Scenario 2 — Hub remote MCP only (team collab)

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

OAuth via `/mcp`. Tools: docs, events, templates, branches, invites.

**Codex CLI (equivalent):** see https://hub.stifer.xyz/codex-setup.md

```bash
codex mcp add hub --url https://hub.stifer.xyz/mcp
codex mcp login hub
```

---

## Optional: both MCPs + soft-sync + namespace bind

Stack agentflow (full bundle) + hub http.

**One workdir / one agentflow namespace ↔ one Hub team (4-char `business_code`, e.g. `z8gw`).**

| Layer | Role |
|-------|------|
| `namespace.metadata["hub.business_code"]` | Product truth for this project |
| `{workdir}/.mycompany/hub-client.json` | Per-repo mirror of code |
| `~/.agent-hub/config.json` | JWT + **fallback** code only (not multi-project truth) |

Resolve order: `env` → **namespace** → workdir → home.

1. Hub OAuth / `hub_login` → JWT in `~/.agent-hub/config.json`  
2. Pick 4-char code from team card / `hub_list_my_businesses` (not display name `zhiji`)  
3. agentflow: `hub_bind_team({ "namespace_id": "<ns>", "business_code": "z8gw" })`  
   - Also accepts paste `zhiji-z8gw` → stores `z8gw`  
4. `hub_status({ "namespace_id": "<ns>" })` → `source=namespace`

Home-only soft-sync config is still useful as JWT storage + fallback; durable multi-project bind is **namespace metadata**.  
Does not replace sticky hooks.
---

## Do not

- MCP without skill or without sticky hooks  
- Leave `YOU` path placeholders  
- Expect Hub login to install `/agentflow on` on another machine  
- Upload SQLite for team sync  
- Default path = `git clone` + `go build` (developers only; see agentflow-setup.md appendix)  
- Continue goals via Bash stdio / JSON-RPC / sqlite when MCP is missing or failed  
