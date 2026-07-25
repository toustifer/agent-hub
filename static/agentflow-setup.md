# agentflow setup for AI agents

> Paste into Claude.  
> **Default install = download Release (no Go, no git clone).**  
> Skill + MCP binary + sticky hooks are mandatory for personal local work on Claude.  
> Hub team MCP is separate.  
> **Codex CLI:** Hub remote MCP + optional agentflow stdio — see https://hub.stifer.xyz/codex-setup.md  
> Updated: 2026-07-25 · Release **v0.2.3**

## Bundle rule (all required for local product)

| Component | Role | Required? |
|-----------|------|-----------|
| **Skill** `~/.claude/skills/agentflow/` | `/agentflow`, flows, hooks (MCP GATE) | **Yes** |
| **MCP** prebuilt binary + stdio | `mcp__agentflow__*` tools | **Yes** |
| **Sticky mode hooks** | `/agentflow on` keeps rules on later turns | **Yes** |
| Hub remote MCP | Team collab tools | Only if team work |
| soft-sync `config.json` | Auto DAG/events to Hub | Optional |

Incomplete if skill / MCP / hooks are missing any one.

## Install (recommended) — one liner

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/toustifer/agentflow/master/scripts/install.sh | bash
```

With auto MCP config write:

```bash
curl -fsSL https://raw.githubusercontent.com/toustifer/agentflow/master/scripts/install.sh \
  | VERSION=v0.2.3 bash -s -- --write-config
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/toustifer/agentflow/master/scripts/install.ps1 | iex
```

Then merge sticky hooks printed by the script into `~/.claude/settings.json` (do **not** wipe other hooks), fully restart Claude Code.

## What the installer does

1. Downloads `skill.tgz` + platform binary from  
   https://github.com/toustifer/agentflow/releases/tag/v0.2.3  
2. Installs to `~/.claude/skills/agentflow/` (+ `bin/agentflow`)  
3. Verifies **MCP GATE** is present in `hooks/mode-lib.js`  
4. Prints (or writes) `mcpServers.agentflow` with `args: ["stdio"]`

## Manual download

| Asset | Platform |
|-------|----------|
| `skill.tgz` | all (required) |
| `agentflow-darwin-arm64` | Apple Silicon |
| `agentflow-darwin-amd64` | Intel Mac |
| `agentflow-linux-amd64` | Linux x64 |
| `agentflow-windows-amd64.exe` | Windows x64 |

Release page: https://github.com/toustifer/agentflow/releases/tag/v0.2.3

### MCP config example (macOS)

```json
{
  "mcpServers": {
    "agentflow": {
      "command": "/Users/YOU/.claude/skills/agentflow/bin/agentflow",
      "args": ["stdio"],
      "type": "stdio"
    }
  }
}
```

### Sticky hooks (**required**)

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

Use absolute paths. Windows: `...\bin\agentflow.exe` and `node C:\Users\YOU\.claude\skills\agentflow\hooks\...`.

---

## Verify (all must pass)

**Three layers — do not mix them up:**

| Layer | Check | Enough for work? |
|-------|--------|------------------|
| Config | `mcpServers.agentflow` in `~/.claude.json` | No |
| UI / process | `/mcp` lists `agentflow` and **not failed** | Required for user |
| **Session tools** | Model can call `mcp__agentflow__flow_ping` this turn | **Yes — only this** |

`claude mcp list` Connected, `agentflow:on` in statusline, or Bash→stdio writing sqlite do **not** count as MCP OK.

1. Fully quit and restart Claude Code  
2. `/mcp` → agentflow **listed and not failed**  
3. In-session: `mcp__agentflow__flow_ping` succeeds  
4. `grep -n "MCP GATE" ~/.claude/skills/agentflow/hooks/mode-lib.js` hits  
5. `/agentflow on` → statusline may show `MCP:cfg|missing|broken`  
6. If MCP tools missing: **stop**; do **not** Bash/JSON-RPC/sqlite around the engine  
7. `/agentflow off` when done  

| Symptom | Fix |
|---------|-----|
| No `/agentflow` | Re-run install / extract `skill.tgz` |
| No tools / `/mcp` failed | Wrong binary path; need `args:["stdio"]`; restart |
| No `MCP GATE` grep | Old skill — reinstall v0.2.3+ |
| Agent uses Bash + stdio | **Invalid** while mode on — fix MCP |

---

## Optional: Hub team MCP + namespace bind

```json
"hub": { "type": "http", "url": "https://hub.stifer.xyz/mcp" }
```

### Soft-sync + bind

**One namespace ↔ one Hub team (4-char code, e.g. `z8gw`).**  
JWT lives in `~/.agent-hub/config.json`; project truth is `namespace.metadata["hub.business_code"]`.

```text
hub_bind_team({ namespace_id, business_code: "z8gw" })  # or "zhiji-z8gw" → z8gw
hub_status({ namespace_id })  # source=namespace
```

Resolve: env → namespace → workdir `.mycompany/hub-client.json` → home.  
See https://hub.stifer.xyz/agent-setup.md
---

## Developer-only (source build)

```bash
git clone https://github.com/toustifer/agentflow.git && cd agentflow
go build -o ~/.claude/skills/agentflow/bin/agentflow ./cmd/agentflow/
# publishers: VERSION=v0.2.3 bash scripts/build-release.sh
```

## Summary

```text
Install agentflow = download skill.tgz + platform binary + sticky hooks
Team tools       = Hub remote MCP (extra)
Auto board       = soft-sync JWT file (extra)
Do NOT          = require go build for end users; Bash-bypass when MCP missing
```

Master: https://hub.stifer.xyz/agent-setup.md  
Repo: https://github.com/toustifer/agentflow  
Release: https://github.com/toustifer/agentflow/releases/tag/v0.2.3
