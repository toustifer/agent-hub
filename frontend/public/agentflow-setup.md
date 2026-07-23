# agentflow setup for AI agents

> Paste into Claude.  
> **Skill + MCP + sticky mode (`/agentflow on`) are a mandatory install for personal local work.**  
> Hub team MCP is separate.  
> Updated: 2026-07-23

## Bundle rule (all required for local product)

| Component | Role | Required? |
|-----------|------|-----------|
| **Skill** `~/.claude/skills/agentflow/` | `/agentflow`, flows, hooks files | **Yes** |
| **MCP** binary + stdio | `mcp__agentflow__*` tools | **Yes** |
| **Sticky mode hooks** | `/agentflow on` keeps rules on later turns | **Yes** |
| Hub remote MCP | Team collab tools | Only if team work |
| soft-sync `config.json` | Auto DAG/events to Hub | Optional |

Incomplete if skill / MCP / hooks are missing any one.

`/agentflow on` is **per machine + per project** (writes `.claude/agentflow/mode.json`). Other devices can have it only after they install the same bundle + hooks.

---

## Install (Windows)

```powershell
git clone https://github.com/toustifer/agentflow.git
cd agentflow

# 1) Skill (includes hooks/)
$dst = "$env:USERPROFILE\.claude\skills\agentflow"
New-Item -ItemType Directory -Force -Path $dst | Out-Null
Copy-Item -Recurse -Force .\skills\agentflow\* $dst

# 2) Binary next to skill
$bin = "$dst\bin"
New-Item -ItemType Directory -Force -Path $bin | Out-Null
go build -o "$bin\agentflow.exe" .\cmd\agentflow\
```

### MCP (`~/.claude.json` or project `.mcp.json`)

```json
{
  "mcpServers": {
    "agentflow": {
      "command": "C:\\Users\\YOU\\.claude\\skills\\agentflow\\bin\\agentflow.exe",
      "args": ["stdio"],
      "type": "stdio"
    }
  }
}
```

### Sticky mode hooks (**required** — enables `/agentflow on`)

Needs **Node.js**. Merge into `~/.claude/settings.json` (do **not** wipe existing hooks):

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node C:\\Users\\YOU\\.claude\\skills\\agentflow\\hooks\\mode-inject.js",
            "timeout": 5
          }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "node C:\\Users\\YOU\\.claude\\skills\\agentflow\\hooks\\statusline.js",
    "refreshInterval": 5
  }
}
```

Replace `YOU` with the real Windows username. Use absolute paths.

---

## Install (macOS / Linux)

```bash
git clone https://github.com/toustifer/agentflow.git
cd agentflow

mkdir -p ~/.claude/skills/agentflow
cp -R skills/agentflow/. ~/.claude/skills/agentflow/

mkdir -p ~/.claude/skills/agentflow/bin
go build -o ~/.claude/skills/agentflow/bin/agentflow ./cmd/agentflow/
```

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

`~/.claude/settings.json` hooks:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node ~/.claude/skills/agentflow/hooks/mode-inject.js",
            "timeout": 5
          }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "node ~/.claude/skills/agentflow/hooks/statusline.js",
    "refreshInterval": 5
  }
}
```

---

## Verify (all must pass)

1. Restart Claude Code  
2. `/mcp` → agentflow **connected**  
3. `/agentflow` → skill loads  
4. `/agentflow on` → mode on (project gets `.claude/agentflow/mode.json`)  
5. `/agentflow status` → enabled  
6. Later normal messages still follow agentflow rules (sticky)  
7. `/agentflow off` when done  

| Symptom | Fix |
|---------|-----|
| No `/agentflow` | Skill not copied |
| No tools | MCP path/binary wrong |
| `on` does nothing / not sticky | hooks missing or wrong path in settings.json |
| No Node | Install Node 18+ for hooks |

---

## Optional: Hub team MCP

```json
"hub": { "type": "http", "url": "https://hub.stifer.xyz/mcp" }
```

### Soft-sync

`hub_export_soft_sync_config` → `~/.agent-hub/config.json`.  
See https://hub.stifer.xyz/agent-setup.md

---

## Summary

```text
Install agentflow = skill + MCP binary + sticky hooks (/agentflow on)
Team tools       = Hub remote MCP (extra)
Auto board       = soft-sync JWT file (extra)
```

Master: https://hub.stifer.xyz/agent-setup.md  
Repo detail: `skills/agentflow/SETUP.md` in https://github.com/toustifer/agentflow
