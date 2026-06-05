# Worker 执行规范（agent-hub）

> 来自 agent-hub 自身的开发经验。每条标注触发条件和来源。

## WKR-101: 写代码前必读上下文

**触发条件**: 任何 worker 任务开始

**规则**:
1. 读 `D:\myprogram\agent-hub\README.md`
2. 读 `D:\myprogram\agent-hub\docs\ARCHITECTURE.md`（重点看自己负责的章节）
3. 读已有代码（不要假设空仓）
4. 读 `.mycompany/leader/leader.json` 看 DAG 当前状态

**来源**: 2026-06-05 立项——避免 worker 重写已有逻辑

---

## WKR-102: 跨模块调用用已定义的接口

**触发条件**: 写 service / handler / 任何 import 其他包的代码

**规则**:
1. 看 leader/leader.json 知道其他 worker 负责什么
2. 看自己能用哪些已生成的 ent 模型（`ent/*.go`）
3. 不要假设方法名——读 `internal/hub/service/*.go` 已写的方法

**来源**: 2026-06-05——避免 worker 重复定义 / 接口名错乱

---

## WKR-103: 不要直接动 ent schema

**触发条件**: 写 service / handler / 前端时遇到"少一个字段"

**规则**:
1. 不要回去改 `ent/schema/*.go`
2. 在 service 层处理（或反馈给 Leader 加 schema）
3. 真要改 schema，停下来报告 Leader，由 worker-database 评估 migration 风险

**来源**: 2026-06-05——schema 变更要重新 generate ent + 写 migration，影响所有 worker

---

## WKR-104: 写完必 commit + 更新 session.json

**触发条件**: 任何 worker 任务完成

**规则**:
1. `git add -A && git commit -m "feat(worker-X): ..."` 在 agent-hub 仓
2. 更新自己 worker dir 的 `session.json`：追加一条 task 记录
3. 有新经验：追加到 `experience.md`
4. 不 commit ＝ 任务不算完成

**来源**: 2026-06-05——保证产物可追溯

---

## WKR-105: 出错立即停 + 报告

**触发条件**: 任何"这不对"的感觉 / 跑不出来的代码 / 不懂的 ent API

**规则**:
1. 立即停手，不要硬试
2. 报告 Leader：现象 + 已经尝试的方案
3. 等 Leader 决策（重派 / 改 brief / 自己 fix）

**来源**: 2026-06-05——避免 worker 在错误方向越走越远
