<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:420px;text-align:center">
      <div v-if="step === 'confirm' && needLogin">
        <div style="font-size:48px;margin-bottom:8px">⚠️</div>
        <h2>请先登录</h2>
        <p style="color:#606266;margin:16px 0">你需要登录 Agent Hub 才能批准设备授权。</p>
        <el-button type="primary" size="large" style="width:100%" @click="goLogin">去登录</el-button>
      </div>
      <div v-else-if="step === 'confirm'">
        <el-icon :size="48" color="#409EFF"><Monitor /></el-icon>
        <h2>{{ $t('device.title') }}</h2>
        <p style="color:#606266;margin:16px 0">{{ $t('device.desc') }}</p>
        <div style="background:#f5f7fa;padding:16px;border-radius:8px;margin-bottom:16px">
          <div style="font-size:24px;font-weight:bold;letter-spacing:4px;color:#303133">{{ code }}</div>
          <div style="color:#909399;font-size:13px;margin-top:4px">{{ $t('device.enterCode') }}</div>
        </div>
        <el-button type="primary" size="large" style="width:100%" @click="approve" :loading="loading">{{ $t('device.approve') }}</el-button>
        <el-button size="large" style="width:100%;margin-top:8px" @click="deny">{{ $t('device.deny') }}</el-button>
      </div>
      <div v-else-if="step === 'done'">
        <el-result icon="success" :title="$t('device.authorized')" :sub-title="$t('device.closePage')" />
        <div v-if="debugInfo" style="margin-top:12px;padding:8px;background:#f5f7fa;border-radius:4px;font-size:12px;color:#909399;word-break:break-all">{{ debugInfo }}</div>
      </div>
      <div v-else>
        <el-result icon="error" :title="$t('device.denied')" :sub-title="$t('device.denyMsg')" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from '@/i18n'
import api from '@/api/hub'

const route = useRoute()
const router = useRouter()
const { t: $t } = useI18n()
const code = ref(route.query.code as string || '')
const manualCode = ref('')
const step = ref('confirm')
const loading = ref(false)
const needLogin = ref(false)

function submitManualCode() {
  if (manualCode.value) {
    code.value = manualCode.value.toUpperCase()
  }
}

onMounted(() => {
  const token = localStorage.getItem('token')
  if (!token) {
    needLogin.value = true
  }
})

function goLogin() {
  const returnUrl = encodeURIComponent('/auth/device?code=' + code.value)
  router.push('/login?redirect=' + returnUrl)
}

const debugInfo = ref('')

async function approve() {
  const token = localStorage.getItem('token')
  if (!token) {
    goLogin()
    return
  }
  loading.value = true
  try {
    debugInfo.value = '正在确认授权...'
    await api.post('/v1/hub/auth/device/confirm?code=' + code.value)
    step.value = 'done'
    const oauthRedirect = route.query.redirect_uri as string
    const oauthState = route.query.state as string
    if (oauthRedirect) {
      const cb = new URL(oauthRedirect)
      cb.searchParams.set('code', code.value)
      if (oauthState) cb.searchParams.set('state', oauthState)
      debugInfo.value = '正在跳转到: ' + cb.toString()
      setTimeout(() => { window.location.href = cb.toString() }, 1000)
      return
    }
    debugInfo.value = '授权成功！正在返回首页...'
    setTimeout(() => { router.push('/') }, 2000)
  } catch (e: any) {
    debugInfo.value = '确认失败: ' + (e?.message || '未知错误')
    loading.value = false
  }
}
function deny() { step.value = 'denied' }
</script>
