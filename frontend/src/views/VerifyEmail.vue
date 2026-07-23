<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:420px;text-align:center">
      <h2 style="margin-bottom:16px">{{ t('verify.title') }}</h2>
      <el-result v-if="status==='ok'" icon="success" :title="t('verify.success')">
        <template #extra>
          <el-button type="primary" @click="goHome">{{ t('verify.enter') }}</el-button>
        </template>
      </el-result>
      <el-result v-else-if="status==='err'" icon="error" :title="t('verify.failed')" :sub-title="error">
        <template #extra>
          <el-button @click="$router.push('/login')">{{ t('login.login') }}</el-button>
        </template>
      </el-result>
      <div v-else>
        <el-icon class="is-loading" :size="32"><i class="el-icon-loading" /></el-icon>
        <p style="margin-top:12px;color:#666">{{ t('verify.working') }}</p>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/i18n'
import { auth } from '@/api/hub'

const route = useRoute()
const router = useRouter()
const store = useAuthStore()
const { t } = useI18n()
const status = ref<'working' | 'ok' | 'err'>('working')
const error = ref('')

onMounted(async () => {
  const token = (route.query.token as string) || ''
  if (!token) {
    status.value = 'err'
    error.value = t('verify.missingToken')
    return
  }
  try {
    const res = await auth.verifyEmail(token)
    const d = res.data.data
    if (d?.token) {
      store.login(d.token, { id: d.user_id, email: d.email })
    }
    status.value = 'ok'
  } catch (e: any) {
    status.value = 'err'
    error.value = e.response?.data?.message || t('login.requestFailed')
  }
})

function goHome() {
  router.push('/app')
}
</script>
