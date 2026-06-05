# Project Context

> agent-hub 的项目背景说明
>
> Created: 2026-06-05

## 定位

agent-company 多租户管理平台（白名单模式）。任何团队的 agent-company
部署可以通过邀请接入 hub，跨租户共享 playbook，单租户内协调 worker。

## 与 sub2api 的关系

- 共享 PostgreSQL（hub schema 隔离）
- 共享 Redis
- 通过 go.mod replace 复用 sub2api 的 AuthService / APIKeyService
- sub2api 0 修改

## 多租户角色

- super-admin：stifer（创建 / 审核 business）
- business-admin：每个 business 的 owner
- worker：通过 apikey 调用受限 API

## 仓库

D:\myprogram\agent-hub（独立 git 仓）
