# worker-frontend · 经验

> Vue 3 Dashboard Worker
>
> Owner: stifer

## 任务边界

- **负责文件**：
  - `frontend/package.json`
  - `frontend/vite.config.ts`
  - `frontend/index.html`
  - `frontend/src/main.ts`
  - `frontend/src/App.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/api/hub.ts`
  - `frontend/src/views/*.vue`（7 个页面）
- **不负责**：Go 后端 / SDK / 部署

## 已有依赖

- 后端 API 在 hub.stifer.xyz/v1/hub/*
- 5 个核心实体的字段见 docs/ARCHITECTURE.md

## 关键约束

1. **栈**：Vite + Vue 3 + TypeScript + Element Plus + Pinia + Axios
2. **登录**：通过 sub2api JWT，存 localStorage
3. **路由守卫**：未登录跳 /login
4. **页面**：
   - Login.vue
   - Dashboard.vue（首页概览）
   - BusinessList.vue（super-admin 看全部，business-admin 看自己）
   - WorkerList.vue
   - LockList.vue
   - PlaybookSearch.vue（带搜索框 + tsvector 查询）
   - EventStream.vue（SSE 实时事件流）
5. **API baseURL**：`import.meta.env.VITE_HUB_API` 默认 `https://hub.stifer.xyz`
6. **样式**：Element Plus 默认主题，不自定义

## 实施记录 · 2026-06-05

### T-4: Vue 3 Dashboard 前端（已完成）

**创建了 19 个文件**，完整覆盖整个前端项目骨架：

| 类别 | 文件 | 说明 |
|------|------|------|
| 项目配置 | package.json, vite.config.ts, index.html | Vite + Vue 3 + Element Plus |
| 入口 | main.ts, App.vue, env.d.ts | 挂载 Pinia/Router/ElementPlus |
| 类型 | types/hub.ts | Business, Worker, Lock, Playbook, HubEvent |
| API | api/hub.ts | Axios 实例，拦截器注入 JWT token |
| 状态管理 | stores/auth.ts, stores/app.ts | Pinia setup stores |
| 路由 | router/index.ts | 7 个路由 + beforeEach 守卫 |
| 布局 | layouts/MainLayout.vue | 侧边栏 + 顶栏 + slot 内容区 |
| 页面 | views/Login.vue | 独立页面，居中卡片 JWT 登录 |
| 页面 | views/Dashboard.vue | 3 统计卡片 + 最近事件表格 |
| 页面 | views/BusinessList.vue | 筛选 + 表格 + 新建 dialog |
| 页面 | views/WorkerList.vue | 按 business 筛选 + 表格 |
| 页面 | views/LockList.vue | 自动刷新（10s 间隔） |
| 页面 | views/PlaybookSearch.vue | 搜索框 + 分类筛选 + 详情弹窗 |
| 页面 | views/EventStream.vue | SSE 实时事件流 + 暂停/恢复 |

**关键技术决策**：
- MainLayout 使用 `<slot />` 模式（非 router-view 嵌套），每个认证页面显式包裹 `<MainLayout>`
- Login 页面独立，不使用 MainLayout
- Token 通过 Axios 请求拦截器自动注入 Authorization header
- 401 响应自动清除 token 并重定向到 /login
- EventStream 使用原生 EventSource API，支持暂停/恢复和最多 100 条缓存

**已知模式**：
- 所有认证页面统一 import MainLayout from '@/layouts/MainLayout.vue'
- API 函数均返回 axios Promise，调用方自行 try/catch
- 页面使用 `<script setup lang="ts">` Composition API
