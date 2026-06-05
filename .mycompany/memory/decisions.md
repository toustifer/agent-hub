# Architecture Decisions（agent-hub）

## D-001: agent-hub 定位为多租户管理平台

**Why**: 用户明确——"agent-hub 是每个人的 agent-company 管理平台"，不是单租户工具
**How to apply**: 5 张表全部带 business_id；路由分 super-admin / business-admin / worker 三层
**Date**: 2026-06-05

## D-002: 白名单模式（仅 super-admin 创建 business）

**Why**: 用户选择"仅受邀"——POC 阶段不开放注册
**How to apply**: hub_businesses.status 初始 'pending'，super-admin 审核改 'active'
**Date**: 2026-06-05

## D-003: 路径 2 部署（独立 Go 服务 + go.mod replace 复用 sub2api）

**Why**: 不改 sub2api 源码，零生产风险
**How to apply**: agent-hub 是独立 git 仓；go.mod 声明 `replace github.com/Wei-Shaw/sub2api => /opt/sub2api-src/backend`
**Date**: 2026-06-05

## D-004: 共享 PostgreSQL + hub schema 隔离

**Why**: 不重搭 DB，sub2api 表完整保留
**How to apply**: 连接串 `search_path=hub,public`；hub_* 表都在 hub schema
**Date**: 2026-06-05

## D-005: 锁用 partial unique index（非 advisory lock）

**Why**: PG 原生支持；acquire = INSERT ON CONFLICT DO NOTHING + SELECT，简洁
**How to apply**: migration 0002 创建 partial unique index；service 层用 SQL 直接操作
**Date**: 2026-06-05

## D-006: playbook 用 PG tsvector（非 SQLite FTS5）

**Why**: 已经在 PG 上；trigger 维护 tsv 列；'simple' 配置对中文友好
**How to apply**: migration 0003 创建 trigger + GIN 索引；service 用 to_tsquery
**Date**: 2026-06-05

## D-007: hub 自带 mixin（不 import sub2api 内部包）

**Why**: 解耦，sub2api 升级不影响 hub
**How to apply**: ent/schema/time_mixin.go + soft_delete_mixin.go；调用时 `Where(hub.DeletedAtIsNil())` 显式过滤
**Date**: 2026-06-05

## D-008: agent-company 框架自用（吃狗粮）

**Why**: 用户明确——"你用 /agent-company 写这个项目，你是 Leader"
**How to apply**: agent-hub 仓下建 .mycompany/ 框架；用 Agent tool 派 worker 子会话
**Date**: 2026-06-05
