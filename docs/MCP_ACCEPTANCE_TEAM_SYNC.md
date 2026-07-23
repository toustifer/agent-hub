# Team collab sync acceptance — 2026-07-22

Host: storyhost `/opt/agent-hub/hub-server` · https://hub.stifer.xyz  
Plan: `docs/plans/2026-07-22-team-collab-sync-plan.md`  
Contract: `docs/SYNC_CONTRACT.md` v0.2

## Migrations applied

| File | Result |
|------|--------|
| `0011_event_actors.sql` | PASS |
| `0012_team_docs.sql` | PASS |
| `0013_worker_templates.sql` | PASS |

## Automated checks (zhiji / JWT)

| Check | Result |
|-------|--------|
| `POST /v1/hub/events` with role+worker | PASS — `actor_user_id`, `actor_email`, `actor_role=worker`, `actor_worker_id` |
| `GET /v1/hub/events?business=zhiji` | PASS — attribution columns present |
| `POST .../docs` upsert team doc | PASS — `updated_by_email` set |
| `GET .../docs` / `GET .../docs/:key` | PASS |
| `POST .../worker-templates` | PASS — published_by_email |
| `GET .../worker-templates` | PASS |
| MCP `tools/list` | PASS — **29 tools** |
| New MCP tools present | PASS — docs + templates + events |
| Public `https://hub.stifer.xyz/mcp` tools/list | PASS — 29 |

## What is now projected (Hub)

1. **Work logs** with company account attribution (user id/email + role + optional worker template id)  
2. **Team docs** (`hub_team_docs`) — timely UPSERT, author fields  
3. **Worker templates** (`hub_worker_templates`) — shareable definitions, not runtime  

Still not projected (by design): SQLite whole DB, local paths, BT/lease, secrets, worker pids.

## MCP tools (29)

Includes prior 24 hub tools plus:

- `hub_upsert_doc` / `hub_list_docs` / `hub_get_doc`  
- `hub_publish_worker_template` / `hub_list_worker_templates`  
- enhanced `hub_append_event` / `hub_list_events`  

## Notes

- Gin: `POST /v1/hub/events` registered once on JWT group (middleware still accepts API key via `tryAPIKey`).  
- agentflow auto soft-sync of lifecycle events remains a follow-up plan (Phase E).  
