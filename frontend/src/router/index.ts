import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'Login', component: () => import('@/views/Login.vue') },
    { path: '/auth/device', name: 'DeviceAuth', component: () => import('@/views/DeviceAuth.vue') },
    { path: '/', name: 'Dashboard', component: () => import('@/views/Dashboard.vue'), meta: { requiresAuth: true } },
    { path: '/team/:code', name: 'TeamPage', component: () => import('@/views/TeamPage.vue'), meta: { requiresAuth: true } },
    { path: '/businesses', name: 'BusinessList', component: () => import('@/views/BusinessList.vue'), meta: { requiresAuth: true } },
    { path: '/workers', name: 'WorkerList', component: () => import('@/views/WorkerList.vue'), meta: { requiresAuth: true } },
    { path: '/locks', name: 'LockList', component: () => import('@/views/LockList.vue'), meta: { requiresAuth: true } },
    { path: '/playbooks', name: 'PlaybookSearch', component: () => import('@/views/PlaybookSearch.vue'), meta: { requiresAuth: true } },
    { path: '/events', name: 'EventStream', component: () => import('@/views/EventStream.vue'), meta: { requiresAuth: true } },
    { path: '/community', name: 'Community', component: () => import('@/views/CommunityPage.vue'), meta: { requiresAuth: true } },
    { path: '/community/:id', name: 'CommunityWorkerDetail', component: () => import('@/views/CommunityWorkerDetail.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router
