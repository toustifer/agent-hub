<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:420px">
      <h2 style="text-align:center;margin-bottom:24px">{{ isRegister ? t('login.createAccount') : t('login.title') }}</h2>

      <template v-if="pendingEmail">
        <el-alert type="success" show-icon :closable="false" style="margin-bottom:16px">
          <template #title>{{ t('login.checkEmail') }}</template>
          <div>{{ t('login.checkEmailHint') }} <b>{{ pendingEmail }}</b></div>
        </el-alert>
        <el-button type="primary" style="width:100%" :loading="resending" @click="resend">
          {{ t('login.resend') }}
        </el-button>
        <el-button type="text" style="width:100%;margin-top:8px" @click="pendingEmail=''">
          {{ t('login.switchToLogin') }}
        </el-button>
        <el-alert v-if="info" :title="info" type="info" show-icon style="margin-top:16px" @close="info=''" />
      </template>

      <template v-else>
        <el-form @submit.prevent="submit">
          <el-form-item><el-input v-model="email" :placeholder="t('login.email')" size="large" /></el-form-item>
          <el-form-item><el-input v-model="password" type="password" :placeholder="t('login.password')" size="large" show-password @keyup.enter="submit" /></el-form-item>
          <el-form-item>
            <el-button type="primary" style="width:100%" size="large" @click="submit" :loading="loading">
              {{ isRegister ? t('login.register') : t('login.login') }}
            </el-button>
          </el-form-item>
        </el-form>
        <el-button type="text" style="width:100%" @click="isRegister = !isRegister; error=''">
          {{ isRegister ? t('login.switchToLogin') : t('login.switchToRegister') }}
        </el-button>
        <el-button v-if="!isRegister && showResend" type="text" style="width:100%" :loading="resending" @click="resend">
          {{ t('login.resend') }}
        </el-button>
        <el-alert v-if="error" :title="error" type="error" show-icon style="margin-top:16px" @close="error=''" />
        <el-alert v-if="info" :title="info" type="info" show-icon style="margin-top:16px" @close="info=''" />
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/i18n'
import { auth } from '@/api/hub'

const router = useRouter()
const route = useRoute()
const store = useAuthStore()
const { t } = useI18n()
const email = ref('')
const password = ref('')
const loading = ref(false)
const resending = ref(false)
const error = ref('')
const info = ref('')
const isRegister = ref(route.query.mode === 'register')
const pendingEmail = ref('')
const showResend = ref(false)

function goApp() {
  const redir = route.query.redirect as string | undefined
  router.push(redir && redir.startsWith('/') ? redir : '/app')
}

async function submit() {
  if (!email.value || !password.value) return
  loading.value = true
  error.value = ''
  info.value = ''
  showResend.value = false
  try {
    if (isRegister.value) {
      const res = await auth.register(email.value, password.value)
      const d = res.data.data
      if (d?.needs_verification) {
        pendingEmail.value = d.email || email.value
        return
      }
      // legacy: if server still returns token
      if (d?.token) {
        store.login(d.token, { id: d.user_id, email: d.email, role: d.role })
        goApp()
      }
    } else {
      const res = await auth.login(email.value, password.value)
      const d = res.data.data
      store.login(d.token, { id: d.user_id, email: d.email, role: d.role })
      goApp()
    }
  } catch (e: any) {
    const data = e.response?.data
    if (data?.error === 'email_not_verified' || data?.code === 403) {
      error.value = data?.message || t('login.notVerified')
      showResend.value = true
      pendingEmail.value = ''
      if (data?.email) email.value = data.email
    } else {
      error.value = data?.message || t('login.requestFailed')
    }
  } finally {
    loading.value = false
  }
}

async function resend() {
  const target = pendingEmail.value || email.value
  if (!target) return
  resending.value = true
  error.value = ''
  info.value = ''
  try {
    await auth.resendVerification(target)
    info.value = t('login.resendOk')
  } catch (e: any) {
    error.value = e.response?.data?.message || t('login.requestFailed')
  } finally {
    resending.value = false
  }
}
</script>
