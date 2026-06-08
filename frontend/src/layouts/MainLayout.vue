<template>
  <el-container style="height: 100vh">
    <el-aside width="220px" class="sidebar">
      <div class="logo"><img src="/logo.png" alt="Agent Hub" style="height:36px" /></div>
      <el-menu :default-active="route.path" router class="sidebar-menu">
        <el-menu-item index="/community"><el-icon><Shop /></el-icon> {{ t('community.title') }}</el-menu-item>
        <el-menu-item index="/"><el-icon><HomeFilled /></el-icon> {{ t('nav.teams') }}</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <span style="font-size:16px">{{ route.name }}</span>
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
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/i18n'
import { Sunny, Moon, HomeFilled, Shop } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { locale, t, setLocale } = useI18n()
const isDark = ref(false)

function toggleLang() { setLocale(locale.value === 'zh' ? 'en' : 'zh') }
function toggleDark(v: boolean) {
  isDark.value = v
  document.documentElement.classList.toggle('dark', v)
  localStorage.setItem('theme', v ? 'dark' : 'light')
}
function handleLogout() { auth.logout(); router.push('/login') }

onMounted(() => {
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
html.dark .topbar { background: #1e1e1e !important; border-bottom-color: #333 !important; color: #e0e0e0; }
html.dark .main-area { background: #141414 !important; }
html.dark .el-card { background: #1e1e1e !important; border-color: #333 !important; color: #e0e0e0; }
.el-table { --el-table-tr-bg-color: #1a1a1a; --el-table-row-hover-bg-color: #2a2a2a; --el-fill-color-blank: #1a1a1a; }
.el-table__row--striped .el-table__cell { background-color: #222 !important; }
html.dark .el-table { --el-table-bg-color: #1e1e1e; --el-table-tr-bg-color: #1e1e1e; --el-table-header-bg-color: #252525; --el-table-border-color: #333; --el-table-text-color: #e0e0e0; --el-table-row-hover-bg-color: #2a2a2a; }
html.dark .el-tag--info { --el-tag-bg-color: #333; --el-tag-text-color: #ccc; }
html.dark .el-tabs__header { border-bottom-color: #333 !important; }
html.dark .el-tabs__item { color: #999 !important; }
html.dark .el-tabs__item.is-active { color: #409EFF !important; }
html.dark .el-timeline-item__node { background: #333 !important; }
html.dark .el-dialog { background: #1e1e1e !important; }
html.dark .el-input__wrapper { background: #252525 !important; box-shadow: 0 0 0 1px #333 inset !important; }
html.dark .el-input__inner { color: #e0e0e0 !important; }
.sidebar { background: #001529; }
.sidebar-menu { border-right: none; }
.logo { padding: 20px 16px; color: #fff; font-size: 18px; font-weight: bold; text-align: center; white-space: nowrap; }
.topbar { background: #fff; border-bottom: 1px solid #e6e6e6; display: flex; align-items: center; justify-content: space-between; padding: 0 20px; }
html.dark .topbar { background: #1e1e1e; }
.main-area { background: #f0f2f5; padding: 20px; }
html.dark .main-area { background: #141414; }
</style>
