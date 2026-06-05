# Leader 工作规范（agent-hub）

> Leader（Claude Code 主会话）调度 worker 子会话时的强制检查项。

## LDR-101: agent-hub 上下文必读

**触发条件**: 任何 agent-hub 任务开始前

**规则**:
1. 必读 `D:\myprogram\agent-hub\README.md`
2. 必读 `D:\myprogram\agent-hub\docs\ARCHITECTURE.md`
3. 必读 `D:\myprogram\agent-hub\docs\PLAN.md`
4. 必读 `.mycompany\leader\leader.json`（DAG 当前状态）

**Why**: agent-hub 任务跨 backend/frontend/deploy/SDK，必须知道全局架构。

---

## LDR-102: 派 worker 必须给完整 brief

**触发条件**: 派 Agent 子会话时

**规则**:
1. Worker 必给：背景、文件路径列表、具体产物、验证标准、不该做什么
2. Worker 完成后必更新：自己 worker dir 的 `session.json` + `experience.md`
3. Worker 产物 commit 进 agent-hub 仓

**Why**: 复用 siruoning 的 Leader 调度经验，避免 worker 不知道边界。

---

## LDR-103: 不亲手写 Phase 2-6 的代码

**触发条件**: Phase 2-6 的具体代码

**规则**:
1. Leader 派 worker 子会话去做
2. Leader 只做：拆任务、看结果、验收、commit
3. 唯一例外：文档、commit message、leader.json 更新

**Why**: 体现"吃自己狗粮"——agent-hub 用 agent-company 开发。

---

## LDR-104: Worker 失败立即回滚 + 重派

**触发条件**: worker 报告任务失败 / 跑偏

**规则**:
1. 立即 git revert worker 的 commit
2. 改 brief 重派，不打补丁

**Why**: 错的方向继续走代价更大。

---

## LDR-105: 验收必须有产物证据

**触发条件**: 验收 worker 任务

**规则**:
1. 必看 worker 改的文件清单（git diff）
2. 必看 worker 的 `session.json` 记录
3. 必看 worker 的 `experience.md` 是否有新经验
4. 必跑 `go build` 验证（如果 worker 没跑过）

**Why**: 减少"我以为我做完了"的盲点。
