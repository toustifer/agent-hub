# Hub Migrations

Migrations live in `internal/hub/repository/migrations/` and are embedded by `migrator.go`.
Versions are filenames without `.sql` (e.g. `0006_auth_foundation`).

## Auth-related (0006+)

| Version | Tables / changes |
|---------|------------------|
| `0006_auth_foundation` | `hub_users`, `hub_memberships`, `hub_api_keys`, `hub_device_codes` |
| `0007_invites_and_join_policy` | `hub_invites`, `hub_businesses.join_policy` |
| `0008_repos_and_branch_index` | `hub_repos`, `hub_branches`, `hub_branch_bindings`, `hub_dag_state` (+ `branch`/`head_sha`) |

### Production note

If production already created auth tables ad-hoc, `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` makes 0006–0007 safe. Before deploy, compare:

```sql
\d hub.hub_users
\d hub.hub_memberships
\d hub.hub_api_keys
\d hub.hub_device_codes
\d hub.hub_businesses
```

### join_policy values

- `invite_only` (default) — need invite token or link-request approval
- `approval` — open join blocked; use link-request
- `open` — anyone with a valid JWT may join by code

Demo projects that relied on open join:

```sql
UPDATE hub.hub_businesses SET join_policy = 'open' WHERE code = 'your-demo-code';
```
