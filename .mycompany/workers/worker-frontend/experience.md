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
