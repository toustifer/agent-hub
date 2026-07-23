# 运维手册

生产：`hub.stifer.xyz` → `storyhost` → `/opt/agent-hub`  
进程：`hub-server`（systemd: `hub-server.service`）→ `:9000`，nginx 反代。

**正规发版流程见 [RELEASE.md](./RELEASE.md)。** 下文为日常运维与应急。

## 服务管理

```bash
ssh storyhost

systemctl status hub-server
systemctl restart hub-server
systemctl stop hub-server
journalctl -u hub-server -f
# 若未挂 unit，进程日志：
tail -f /var/log/agent-hub.log
```

## 健康检查

```bash
curl -sS https://hub.stifer.xyz/health
curl -sS https://hub.stifer.xyz/version
curl -sS https://hub.stifer.xyz/healthz
```

`/health` 与 `/version` 含 `version` / `commit` / `build_time`（ldflags 注入）。

## 部署（推荐）

```bash
# 1) 构建
./scripts/build-release.sh v0.2.0

# 2) 部署
./scripts/deploy-release.sh v0.2.0 storyhost
```

或：推送 git tag `v0.2.0` → GitHub Actions 产出 Release 附件 → 本机下载后 `deploy-release.sh`。

## 监控

| 指标 | 来源 | 阈值 |
|------|------|------|
| 进程 | `systemctl is-active hub-server` | active |
| 版本 | `curl /version` | 与预期 tag 一致 |
| 内存 | `ps aux \| grep hub-server` | 视负载 |
| 锁 | `SELECT count(*) FROM hub.hub_locks WHERE …` | 异常堆积 |
| DB | `pg_stat_activity` | 连接打满 |

## 备份

```bash
# hub schema
pg_dump "$HUB_DATABASE_URL" -n hub | gzip > /opt/backup/hub_$(date +%F).sql.gz
```

## 故障恢复

### 进程挂了

```bash
journalctl -u hub-server -n 100
# 或
tail -100 /var/log/agent-hub.log
systemctl restart hub-server
curl -sS http://127.0.0.1:9000/health
```

### 回滚二进制

```bash
cd /opt/agent-hub
ls -lt hub-server.bak.* | head
cp -a hub-server.bak.YYYYMMDDHHMMSS hub-server
systemctl restart hub-server
```

或：`install -m 755 releases/vX.Y.Z/hub-server hub-server && systemctl restart hub-server`

### 锁未释放

等 TTL，或管理员 SQL 标记释放（谨慎）。

## 安全

- `.env` 仅在服务器，勿进 git  
- 定期轮换 JWT secret / SMTP 授权码 / API Key  
- 审计：`hub.hub_events`  

## 与旧文档差异

旧文中的 `sub2api-hub` 单元名已废弃，现用 **`hub-server`**。  
路径 `/opt/agent-hub`，二进制名 **`hub-server`**。
