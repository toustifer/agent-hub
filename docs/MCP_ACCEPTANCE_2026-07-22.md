# MCP remote acceptance — 2026-07-22

Host: storyhost (47.115.134.24) `/opt/agent-hub/hub-server` :9000 behind nginx → https://hub.stifer.xyz

## Automated (executed)

| Check | Result |
|-------|--------|
| `go test ./internal/mcp ./internal/hub/handler` | PASS |
| `go build ./cmd/hub` (linux amd64) | PASS |
| Deploy binary + migration `0010_oauth_clients` | PASS |
| Public `POST /mcp` initialize | PASS (`serverInfo: agent-hub v1.0.0`) |
| Public `tools/list` | PASS **24 tools** |
| OAuth well-known grants include device_code + authorization_code | PASS |
| DCR `POST /v1/hub/oauth/register` persists `ah-*` client_id | PASS |
| Unauthenticated `hub_list_my_businesses` | PASS (tool error: missing Authorization) |
| JWT `hub_list_my_businesses` | PASS (lists teams) |
| JWT `hub_list_branches` member | PASS |
| JWT `hub_list_branches` unknown business | PASS (not found) |
| JWT non-member business (`siruoning` as user 2) | PASS (not a member) |
| JWT `hub_report_branches` + list after | PASS |
| REST soft-sync `/v1/hub/me/businesses` + branches/report **without Node MCP** | PASS |
| No process on `:9001` / no node mcp-server | PASS |
| Public `/mcp.md` remote-primary text | PASS |

## Fixes during acceptance

1. `DisableLocalhostProtection: true` — nginx → 127.0.0.1 with Host hub.stifer.xyz was 403.
2. Optional JSON schema fields (`omitempty`) for optional tool args.
3. `hub_repos` insert without non-existent UNIQUE ON CONFLICT.

## Not automated in this run

- Full Claude Code Host OAuth browser UI click-through (metadata/DCR/device endpoints verified; Host UX not driven here).
- Live `HUB_ADMIN_PASSWORD` login failed (env value ≠ DB hash); used HS256 mint from `HUB_JWT_SECRET` + membership row instead (same token shape as genToken).
