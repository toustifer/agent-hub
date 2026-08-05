# Changelog

All notable changes to agent-hub are documented here.
Format inspired by Keep a Changelog. Versioning: [SemVer](https://semver.org/).

## [Unreleased]

### Added
- Formal release pipeline: `VERSION`, `scripts/build-release.sh`, `scripts/deploy-release.sh`
- GitHub Actions: `ci.yml` (test), `release.yml` (tag → artifacts + GitHub Release)
- `/version` and version fields on `/health`
- Docs: `docs/RELEASE.md`

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

[Unreleased]: https://github.com/toustifer/agent-hub/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.2
[0.2.0]: https://github.com/toustifer/agent-hub/releases/tag/v0.2.0
