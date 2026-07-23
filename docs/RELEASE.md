# 发版与部署

> 目标：用 **semver tag + 可复现产物 + 一键部署** 替代手工 scp。  
> 生产：`storyhost` → `/opt/agent-hub`，systemd `hub-server.service`，nginx → `:9000`。

## 版本号

- 源文件：`VERSION`（如 `0.2.0`，不含 `v`）
- Git tag：`v0.2.0`（带 `v`）
- 二进制内嵌：`/version`、`/health` → `version.Version` / commit / build_time
- MCP `serverInfo.version` 同源

## 常规发版流程

### 1. 准备

```bash
cd D:\myprogram\agent-hub   # or your clone
git checkout main
git pull
# 确认迁移：internal/hub/repository/migrations/
#  bump VERSION 文件
echo 0.2.0 > VERSION
```

### 2. 打 tag 并推送（触发 CI Release）

```bash
git add VERSION
git commit -m "chore: release v0.2.0"
git tag -a v0.2.0 -m "agent-hub v0.2.0"
git push origin main
git push origin v0.2.0
```

GitHub Actions `release.yml` 会：

1. `go test ./internal/...`
2. `scripts/build-release.sh v0.2.0` → `dist/release/agent-hub-v0.2.0-linux-amd64.tar.gz`
3. 上传 Artifact + 创建 GitHub Release（附件为 tar.gz）

### 3. 部署到生产

在有 `storyhost` SSH 的机器上：

```bash
# 若本地已有产物
./scripts/build-release.sh v0.2.0   # 可选，或从 Release 下载 tar.gz 到 dist/release/
./scripts/deploy-release.sh v0.2.0 storyhost
```

`deploy-release.sh` 会：

1. 上传 tar 到 `/opt/agent-hub/releases/`
2. 解压 → `releases/v0.2.0/`
3. 按序 apply `migrations/*.sql`（`IF NOT EXISTS` 安全）
4. 备份旧 `hub-server` → `hub-server.bak.<timestamp>`
5. 安装新二进制 + 同步 `static/`
6. `systemctl restart hub-server`（若单元存在）
7. `curl localhost:9000/health` + `/version`

### 4. 验收

```bash
curl -sS https://hub.stifer.xyz/health | jq .
curl -sS https://hub.stifer.xyz/version | jq .
# 期望 version 字段为 v0.2.0
```

### 5. 回滚

```bash
ssh storyhost
cd /opt/agent-hub
# 列备份
ls -lt hub-server.bak.* | head
# 恢复
cp -a hub-server.bak.YYYYMMDDHHMMSS hub-server
systemctl restart hub-server
# 或切回旧 release 目录
# install -m 755 releases/v0.1.0/hub-server hub-server && systemctl restart hub-server
```

Schema 回滚需单独 SQL（正向 migration 默认不自动 down）。

## 本地不经 GitHub 的热修

```bash
./scripts/build-release.sh v0.2.1-hotfix
./scripts/deploy-release.sh v0.2.1-hotfix storyhost
```

仍建议随后补 tag 与 Release 记录。

## 环境变量

生产 `/opt/agent-hub/.env`（**勿进 git**）。样例见 `.env.example`。

部署脚本读取远端 `.env` 里的 `HUB_DATABASE_URL` 跑 migration。

## 产物布局

```text
dist/release/v0.2.0/
  hub-server          # linux/amd64
  static/             # vite dist + public md
  migrations/         # *.sql
  RELEASE.txt
dist/release/agent-hub-v0.2.0-linux-amd64.tar.gz
```

## 与旧流程对照

| 旧 | 新 |
|----|-----|
| 本机 `go build` + scp 单文件 | `build-release.sh` 打 tar |
| 手写 migration 命令 | deploy 脚本批量 apply |
| `nohup` 手动重启 | 优先 `systemctl restart hub-server` |
| 无版本可见性 | `/version` + health 内嵌 version |
| bak 文件一堆 | release 目录 + bak 时间戳仍保留 |

## agentflow 客户端

agentflow **不随 Hub Release 自动升级**；各机自行：

```bash
git pull && go build -o agentflow ./cmd/agentflow/
```

Hub 与 agentflow 用 **API 契约** 对齐（见 `SYNC_CONTRACT.md`），semver 大版本破坏时在 Release notes 标明。
