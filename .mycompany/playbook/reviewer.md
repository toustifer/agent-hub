# Reviewer 规范（agent-hub）

> Leader 验收 worker 任务的检查清单。

## REV-101: 文件清单核对

**规则**:
1. 看 worker brief 里要求的文件是否都创建 / 修改
2. 不在 brief 里的文件被改 → 报告异常

## REV-102: 编译验证

**规则**:
1. worker 自己没跑过 `go build`（可能没 Go 环境）→ Leader 跑一次
2. go.mod 引用是否对（ent 0.14.5 / gin 1.10 / pgx5）

## REV-103: 日志体系核对

**规则**:
1. worker 自己的 `session.json` 必更新
2. 重要决策进 `decisions.md`
3. 经验进 `experience.md`

## REV-104: 跨 worker 一致性

**规则**:
1. service 暴露的方法名，handler 调用时要对得上
2. ent 生成的模型，service / handler / 前端都要对得上
3. 路由表 router.go vs handler 实际注册
