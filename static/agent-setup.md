# Agent setup for AI (Hub MCP + agentflow)

> Paste this whole file into Claude / Cursor.  
> **Codex CLI users:** prefer https://hub.stifer.xyz/codex-setup.md (same Hub, different host config).  
> Base: https://hub.stifer.xyz  
> Updated: 2026-07-23

## Critical: agentflow install is a three-part bundle

**Personal local work requires ALL of:**

1. **Skill** — `/agentflow`, flows (`~/.claude/skills/agentflow/`)  
2. **MCP** — stdio binary (`mcp__agentflow__*`)  
3. **Sticky mode hooks** — `/agentflow on` stays on for later turns (`mode-inject.js` in settings.json)

| Missing | Result |
|---------|--------|
| No skill | No `/agentflow` orchestration |
| No MCP | No tools |
| No hooks | `/agentflow on` does not stick; product incomplete |
| **All three** | Intended install |

Do **not** treat “only `.mcp.json` → agentflow.exe” as complete.

---

## Two scenarios

| Scenario | Need |
|----------|------|
| **1. Personal local work** | **Skill + MCP + sticky hooks** (must) |
| **2. Team collab only** | Hub remote MCP only |
| **1 + 2** | Bundle + Hub MCP; soft-sync optional via config file |

---

## Scenario 1 — full agentflow install (required for personal work)

### 1) Clone

```bash
git clone https://github.com/toustifer/agentflow.git
cd agentflow
```

### 2) Install skill (required)

**Windows:**

```powershell
$dst = "$env:USERPROFILE\.claude\skills\agentflow"
New-Item -ItemType Directory -Force -Path $dst | Out-Null
Copy-Item -Recurse -Force .\skills\agentflow\* $dst
```

**macOS / Linux:**

```bash
mkdir -p ~/.claude/skills/agentflow
cp -R skills/agentflow/. ~/.claude/skills/agentflow/
```

### 3) Build binary (required)

**Windows:**

```powershell
$bin = "$env:USERPROFILE\.claude\skills\agentflow\bin"
New-Item -ItemType Directory -Force -Path $bin | Out-Null
go build -o "$bin\agentflow.exe" .\cmd\agentflow\
```

**macOS / Linux:**

```bash
mkdir -p ~/.claude/skills/agentflow/bin
go build -o ~/.claude/skills/agentflow/bin/agentflow ./cmd/agentflow/
```

### 4) MCP config (required)

**Windows `~/.claude.json` / project `.mcp.json`:**

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

### 5) Sticky mode hooks — `/agentflow on` (required)

Needs **Node.js 18+**. Merge into `~/.claude/settings.json` (append, do not erase other hooks):

**Windows:**

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

**macOS / Linux:** use `node ~/.claude/skills/agentflow/hooks/mode-inject.js` (and statusline.js).

### 6) Verify

1. Restart Claude  
2. `/mcp` → agentflow connected  
3. `/agentflow` → skill  
4. `/agentflow on` → sticky on  
5. `/agentflow status` → enabled  
6. Normal follow-up messages still use agentflow rules  

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

## Optional: both MCPs + soft-sync

Stack agentflow (full bundle) + hub http.  

Soft-sync: after Hub OAuth, `hub_export_soft_sync_config({ "business_code": "<4-char team code>" })` → write `~/.agent-hub/config.json`.  
`business_code` is the **auto-generated short code** shown on the team card (not the display name, not the full `/team/name-code` path).  
Old hand-filled codes work for ~90 days via alias after migration.  
Does not replace sticky hooks.

---

## Do not

- MCP without skill or without sticky hooks  
- Leave `ABS_PATH` / `YOU` placeholders  
- Expect Hub login to install `/agentflow on` on another machine  
- Upload SQLite for team sync  
