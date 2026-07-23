<template>
  <div class="auth-shell">
    <div class="glow glow-a" />
    <div class="glow glow-b" />

    <div class="auth-card">
      <div class="brand">
        <div class="logo">AH</div>
        <div>
          <div class="brand-name">Agent Hub</div>
          <div class="brand-sub">{{ $t('device.brandSub') }}</div>
        </div>
      </div>

      <!-- need login -->
      <div v-if="step === 'confirm' && needLogin" class="panel">
        <div class="icon-wrap warn">!</div>
        <h1>{{ $t('device.needLoginTitle') }}</h1>
        <p class="lead">{{ $t('device.needLoginDesc') }}</p>
        <button class="btn primary" type="button" @click="goLogin">{{ $t('device.goLogin') }}</button>
      </div>

      <!-- approve -->
      <div v-else-if="step === 'confirm'" class="panel">
        <div class="icon-wrap">
          <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="1.8">
            <rect x="3" y="4" width="18" height="14" rx="2" />
            <path d="M8 21h8M12 18v3" />
          </svg>
        </div>
        <h1>{{ $t('device.title') }}</h1>
        <p class="lead">{{ $t('device.desc') }}</p>

        <div class="code-box">
          <div class="code-label">{{ $t('device.codeLabel') }}</div>
          <div class="code-value">{{ code || '————' }}</div>
          <div class="code-hint">{{ $t('device.enterCode') }}</div>
        </div>

        <button class="btn primary" type="button" :disabled="loading || !code" @click="approve">
          <span v-if="loading" class="spinner" />
          {{ $t('device.approve') }}
        </button>
        <button class="btn ghost" type="button" :disabled="loading" @click="deny">{{ $t('device.deny') }}</button>
        <p v-if="debugInfo" class="debug">{{ debugInfo }}</p>
      </div>

      <!-- success -->
      <div v-else-if="step === 'done'" class="panel">
        <div class="icon-wrap success">
          <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="2.2">
            <path d="M5 12.5l4.5 4.5L19 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </div>
        <h1>{{ $t('device.authorized') }}</h1>
        <p class="lead">{{ $t('device.closePage') }}</p>
        <div class="success-chip">Claude Code · MCP</div>
        <p class="footnote">{{ $t('device.successFoot') }}</p>
        <p v-if="debugInfo" class="debug">{{ debugInfo }}</p>
      </div>

      <!-- denied -->
      <div v-else class="panel">
        <div class="icon-wrap danger">×</div>
        <h1>{{ $t('device.denied') }}</h1>
        <p class="lead">{{ $t('device.denyMsg') }}</p>
        <button class="btn ghost" type="button" @click="backToConfirm">
          {{ $t('device.back') }}
        </button>
      </div>
    </div>
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
const code = ref((route.query.code as string) || '')
const step = ref<'confirm' | 'done' | 'denied'>('confirm')
const loading = ref(false)
const needLogin = ref(false)
const debugInfo = ref('')

onMounted(() => {
  if (!localStorage.getItem('token')) {
    needLogin.value = true
  }
})

function goLogin() {
  const returnUrl = encodeURIComponent('/auth/device?code=' + code.value + locationSearchExtra())
  router.push('/login?redirect=' + returnUrl)
}

function backToConfirm() {
  step.value = 'confirm'
  needLogin.value = !localStorage.getItem('token')
  debugInfo.value = ''
}

function locationSearchExtra() {
  const q = new URLSearchParams()
  const r = route.query.redirect_uri as string
  const s = route.query.state as string
  if (r) q.set('redirect_uri', r)
  if (s) q.set('state', s)
  const str = q.toString()
  return str ? '&' + str : ''
}

async function approve() {
  const token = localStorage.getItem('token')
  if (!token) {
    goLogin()
    return
  }
  if (!code.value) {
    debugInfo.value = $t('device.missingCode')
    return
  }
  loading.value = true
  debugInfo.value = ''
  try {
    await api.post('/v1/hub/auth/device/confirm?code=' + encodeURIComponent(code.value))
    step.value = 'done'
    const oauthRedirect = route.query.redirect_uri as string
    const oauthState = route.query.state as string
    if (oauthRedirect) {
      try {
        const cb = new URL(oauthRedirect)
        cb.searchParams.set('code', code.value)
        if (oauthState) cb.searchParams.set('state', oauthState)
        debugInfo.value = $t('device.redirecting')
        setTimeout(() => { window.location.href = cb.toString() }, 900)
        return
      } catch {
        debugInfo.value = $t('device.badRedirect')
      }
    }
  } catch (e: any) {
    const status = e?.response?.status
    const msg = e?.response?.data?.message || e?.message || 'error'
    if (status === 401) {
      debugInfo.value = $t('device.tokenInvalid')
      needLogin.value = true
    } else if (status === 404) {
      debugInfo.value = $t('device.codeInvalid')
    } else {
      debugInfo.value = msg
    }
  } finally {
    loading.value = false
  }
}

function deny() {
  step.value = 'denied'
}
</script>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  position: relative;
  overflow: hidden;
  background: #0b1020;
  color: #e8eefc;
  font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif;
}
.glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  opacity: 0.55;
}
.glow-a {
  width: 420px;
  height: 420px;
  background: #3b82f6;
  top: -120px;
  left: -80px;
  opacity: 0.25;
}
.glow-b {
  width: 380px;
  height: 380px;
  background: #8b5cf6;
  bottom: -140px;
  right: -60px;
  opacity: 0.22;
}
.auth-card {
  position: relative;
  width: 100%;
  max-width: 440px;
  background: linear-gradient(160deg, rgba(26, 32, 56, 0.95), rgba(15, 23, 42, 0.98));
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: 20px;
  padding: 28px 28px 32px;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.45), inset 0 1px 0 rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(12px);
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
  padding-bottom: 18px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}
.logo {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  font-weight: 800;
  font-size: 14px;
  letter-spacing: 0.5px;
  color: #fff;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.35);
}
.brand-name {
  font-size: 16px;
  font-weight: 700;
  color: #f8fafc;
}
.brand-sub {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 2px;
}
.panel {
  text-align: center;
}
.icon-wrap {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  border-radius: 18px;
  display: grid;
  place-items: center;
  color: #93c5fd;
  background: rgba(59, 130, 246, 0.12);
  border: 1px solid rgba(59, 130, 246, 0.25);
}
.icon-wrap.success {
  color: #6ee7b7;
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.3);
}
.icon-wrap.warn {
  color: #fcd34d;
  background: rgba(245, 158, 11, 0.12);
  border-color: rgba(245, 158, 11, 0.3);
  font-size: 28px;
  font-weight: 800;
}
.icon-wrap.danger {
  color: #fca5a5;
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.3);
  font-size: 32px;
  font-weight: 700;
  line-height: 1;
}
h1 {
  margin: 0 0 10px;
  font-size: 22px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: -0.02em;
}
.lead {
  margin: 0 0 22px;
  font-size: 14px;
  line-height: 1.65;
  color: #94a3b8;
}
.code-box {
  background: rgba(15, 23, 42, 0.75);
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 14px;
  padding: 16px 14px 14px;
  margin-bottom: 20px;
}
.code-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #64748b;
  margin-bottom: 8px;
}
.code-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0.28em;
  color: #e0f2fe;
  padding-left: 0.28em;
}
.code-hint {
  margin-top: 10px;
  font-size: 12px;
  color: #64748b;
}
.btn {
  width: 100%;
  border: none;
  border-radius: 12px;
  padding: 13px 16px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.12s ease, opacity 0.12s ease, box-shadow 0.12s ease;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}
.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.btn.primary {
  color: #fff;
  background: linear-gradient(135deg, #3b82f6, #6366f1);
  box-shadow: 0 10px 28px rgba(59, 130, 246, 0.35);
  margin-bottom: 10px;
}
.btn.primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 14px 32px rgba(59, 130, 246, 0.42);
}
.btn.ghost {
  color: #cbd5e1;
  background: transparent;
  border: 1px solid rgba(148, 163, 184, 0.25);
}
.btn.ghost:hover:not(:disabled) {
  background: rgba(148, 163, 184, 0.08);
}
.success-chip {
  display: inline-block;
  margin: 0 auto 14px;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: #a5b4fc;
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid rgba(99, 102, 241, 0.28);
}
.footnote {
  margin: 0;
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
}
.debug {
  margin-top: 14px;
  font-size: 12px;
  color: #94a3b8;
  word-break: break-all;
  line-height: 1.45;
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.35);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
