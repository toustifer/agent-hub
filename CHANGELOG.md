# Changelog

All notable changes to agent-hub are documented here.
Format inspired by Keep a Changelog. Versioning: [SemVer](https://semver.org/).

## [Unreleased]

### Added
- Formal release pipeline: `VERSION`, `scripts/build-release.sh`, `scripts/deploy-release.sh`
- GitHub Actions: `ci.yml` (test), `release.yml` (tag → artifacts + GitHub Release)
- `/version` and version fields on `/health`
- Docs: `docs/RELEASE.md`

## [0.2.5] - 2026-10-02

### Removed
- Delete legacy root migrations/ directory; migrations are exclusively sourced from internal/hub/repository/migrations/*.sql via embed.
- Remove obsolete and unrouted McpSetup.vue view (superseded by OfficialDocs.vue).
- Remove unused Pinia pp.ts store.
- Remove unreferenced mcp.* translation strings from rontend/src/i18n/index.ts.
- Remove unreferenced private deidentify function in community_service.go.
- Remove empty placeholder if block in doc_handler.go.
- Delete one-off maintenance scripts in scripts/ and ignore temporary static documentation build artifacts in .gitignore.

### Refactored
- Deduplicate team path helper in usiness_handler.go by directly calling service.BuildTeamPath.

## [0.2.4] - 2026-10-02

### Fixed
- Align Kanban board task columns in KanbanBoard.vue with native gentflow states (ssigned -> pending, xecuting/ework_needed -> in_progress, eview_pending -> in_review, done/passed -> completed).
- Display status tag badge directly on kanban cards for improved visual feedback.
- Update 	askStatusType in TeamPage.vue to map all gentflow task states to appropriate tag types (done -> success, xecuting -> warning, ework_needed -> danger, eview_pending -> primary).
- Update 	asks_done subquery in equirement_handler.go (ListRequirements & GetRequirement) from hardcoded d.status = 'completed' to d.status IN ('completed', 'done', 'passed'), resolving 0-completion count issue for agentflow tasks.

## [0.2.3] - 2026-09-18

### Security
- Enforce team membership check in `InstallWorker` to prevent unauthorized worker installation into other tenants.
- Enforce membership verification on `ListWorkers` and `ListActiveLocks` when `business` parameter is supplied; disallow empty query to prevent cross-tenant enumeration.
- Guard against nil pointer and type assertion panic in `ReviewLinkRequest`.

### Fixed
- Fix SQL parameter placeholder collision in `UpdateRequirement` when updating both `title` and `description`.
- Add database migration `0017_playbooks_unique_constraint.sql` adding `UNIQUE(business_id, category, title)` to `hub_playbooks` for `ON CONFLICT` support.
- Fix scope slot destructuring in `TeamPage.vue` DAG status column (`{row}` instead of `{r}`).
- Replace missing `StatusBadge` component with Element Plus `el-tag` in `RequirementsTab.vue`.
- Decouple hardcoded domain in `TeamPage.vue` SSE connection to support `VITE_HUB_API` and relative paths.
- Align `setup.html` with official Streamable HTTP MCP endpoint.
- Copy all `frontend/public/*` documents and assets during release packaging in `build-release.sh`.

## [0.2.2] - 2026-08-05

### Added
- Requirements, comments, task linking, and status transitions for non-technical collaboration.
- Matching Hub REST routes and leader-agent MCP tools for requirements.
- Federation dashboard components and the refreshed documentation portal.
- Login support for HTTP Hub URLs and email-based authentication.
- Frontend TypeScript project configuration so CI type-checks before bundling.

### Changed
- Updated Hub navigation and portal layout for the federation workflow.
- Added the `0016_requirements` database migration.

## [0.2.0] - 2026-07-23

### Added
- Remote MCP (Streamable HTTP), team collab sync (events attribution, team docs, worker templates)
- Official docs tab (Hub MCP + agentflow setup)
- Soft-sync export tool `hub_export_soft_sync_config`

[Unreleased]: https://github.com/toustifer/agent-hub/compare/v0.2.5...HEAD
[0.2.5]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.5
[0.2.4]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.4
[0.2.3]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.3
[0.2.2]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.2
[0.2.0]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.0
