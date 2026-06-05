# 运维手册

## 服务管理

agent-hub 作为 systemd 单元运行：

```bash
# 状态
systemctl status sub2api-hub

# 启动 / 停止 / 重启
sudo systemctl start sub2api-hub
sudo systemctl stop sub2api-hub
sudo systemctl restart sub2api-hub

# 日志
journalctl -u sub2api-hub -f
tail -f /var/log/sub2api-hub/*.log
```

## 部署

```bash
# 本地构建（需先有 /opt/sub2api-src/backend）
cd D:\myprogram\agent-hub
go build -o bin/hub ./cmd/hub

# 上传到服务器
scp bin/hub root@47.115.134.24:/opt/agent-hub/

# 服务器上
sudo systemctl restart sub2api-hub
```

## 监控

| 指标 | 来源 | 阈值 |
|---|---|---|
| 进程内存 | `ps aux \| grep hub` | < 200MB |
| CPU | `top -p $(pgrep hub)` | < 50% 平均 |
| 锁活跃数 | `SELECT count(*) FROM hub.hub_locks WHERE released_at IS NULL AND expires_at > now()` | < 100 |
| 死 Worker | `SELECT count(*) FROM hub.hub_workers WHERE status = 'offline' AND last_heartbeat_at < now() - interval '5 min'` | < 5 |
| DB 连接 | pg_stat_activity | < 50 |

## 备份

```bash
# 每日备份 hub schema
pg_dump -U sub2api -n hub sub2api | gzip > /opt/backup/hub_$(date +%F).sql.gz

# 恢复
gunzip -c /opt/backup/hub_2026-06-05.sql.gz | psql -U sub2api sub2api
```

## 升级

```bash
# 1. 拉新代码
cd D:\myprogram\agent-hub
git pull

# 2. 跑 migration（如有）
psql -U sub2api -d sub2api -f migrations/000X_*.sql

# 3. 重新生成 ent（如 schema 变了）
go generate ./ent

# 4. 重新构建
go build -o bin/hub ./cmd/hub
scp bin/hub root@47.115.134.24:/opt/agent-hub/

# 5. 重启
ssh root@47.115.134.24 'systemctl restart sub2api-hub'

# 6. 冒烟
curl https://hub.stifer.xyz/health
```

## 故障恢复

### Hub 挂了
```bash
# 1. 看日志
journalctl -u sub2api-hub -n 100

# 2. 常见原因：
#    - DB 连接耗尽：重启 sub2api / 检查 pg 连接池
#    - Redis 挂了：重启 redis-server
#    - 二进制坏：scp 重新上传
#    - 端口被占：lsof -i :9000

# 3. 紧急回滚
sudo systemctl stop sub2api-hub
scp /opt/backup/hub.bak root@47.115.134.24:/opt/agent-hub/hub
sudo systemctl start sub2api-hub
```

### 锁死（Worker 崩溃没释放）
```bash
# 等 5 分钟 TTL 自动过期
# 或者 admin 强制释放
psql -U sub2api -d sub2api -c \
  "UPDATE hub.hub_locks SET released_at = now() WHERE id = $1"
```

### 死锁（两个 Worker 互相等）
agent-hub 不支持嵌套锁，hub-boot 会拒绝。出现死锁要查 hub_event 找出冲突链。

## 安全

- 不要把 admin JWT 写进仓库
- APIKey 限定 scope（`hub:write` / `hub:read`）
- 定期 rotate APIKey（每 6 个月）
- 看 audit log：`SELECT * FROM hub.hub_events ORDER BY created_at DESC LIMIT 100`
- 监控异常来源 IP

## 性能

- 单实例 30-50MB 内存
- 100 并发心跳 < 50ms p99
- 锁 acquire < 20ms p99
- 全文搜索 < 100ms p99（10 万 playbook 量级）

不够用时考虑：
- 水平扩展：多个 agent-hub 进程（同 DB，PG 锁保证一致）
- 读写分离：playbook 读走从库
- CDN：dashboard 静态资源走 CDN
