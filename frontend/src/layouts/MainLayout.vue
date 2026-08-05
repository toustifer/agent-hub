<template>
  <el-container style="height: 100vh">
    <el-aside width="220px" class="sidebar">
      <div class="logo"><img src="/logo.png" alt="Agent Hub" style="height:36px" /></div>
      <el-menu :default-active="activeMenu" router class="sidebar-menu">
        <el-menu-item index="/community"><el-icon><Shop /></el-icon> {{ t('community.title') }}</el-menu-item>
        <el-menu-item index="/app"><el-icon><HomeFilled /></el-icon> {{ t('nav.teams') }}</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <span style="font-size:16px">{{ pageTitle }}</span>
        <div style="display:flex;align-items:center;gap:12px">
          <span style="color:#909399;font-size:13px">{{ auth.user?.email }}</span>
          <el-button size="small" circle @click="toggleLang">{{ locale === 'zh' ? 'EN' : '中' }}</el-button>
          <el-switch v-model="isDark" @change="toggleDark" :active-icon="Moon" :inactive-icon="Sunny" inline-prompt />
          <el-button type="danger" size="small" @click="handleLogout">{{ t('nav.logout') }}</el-button>
        </div>
      </el-header>
      <el-main class="main-area">
        <slot />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/i18n'
import { Sunny, Moon, HomeFilled, Shop } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { locale, t, setLocale } = useI18n()
const isDark = ref(false)

const activeMenu = computed(() => {
  if (route.path === '/community' || route.path.startsWith('/community/')) return '/community'
  if (route.path.startsWith('/team/') || route.path === '/app' || route.path === '/dashboard') return '/app'
  return route.path
})

const pageTitle = computed(() => {
  if (activeMenu.value === '/community') return t('community.title')
  if (activeMenu.value === '/app') return t('nav.teams')
  return String(route.name || '')
})

function toggleLang() { setLocale(locale.value === 'zh' ? 'en' : 'zh') }
function toggleDark(v: boolean) {
  isDark.value = v
  document.documentElement.classList.toggle('dark', v)
  localStorage.setItem('theme', v ? 'dark' : 'light')
}
function handleLogout() { auth.logout(); router.push('/login') }

onMounted(() => {
  document.title = 'Agent Hub'
  const saved = localStorage.getItem('theme')
  if (saved === 'dark' || (!saved && window.matchMedia('(prefers-color-scheme:dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
})
</script>

<style>
html.dark { color-scheme: dark; }
html.dark body { background: #141414; }
html.dark .sidebar { background: #1e1e1e !important; }
html.dark .sidebar-menu { background: #1e1e1e !important; border-right-color: #333 !important; }
html.dark .el-menu-item { color: #999 !important; }
html.dark .el-menu-item:hover { background: #2a2a2a !important; }
html.dark .el-menu-item.is-active { color: #409EFF !important; }
</style>
