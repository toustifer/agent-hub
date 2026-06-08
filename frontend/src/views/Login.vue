<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:400px">
      <h2 style="text-align:center;margin-bottom:24px">{{ isRegister ? t('login.createAccount') : t('login.title') }}</h2>
      <el-form @submit.prevent="submit">
        <el-form-item><el-input v-model="email" :placeholder="t('login.email')" size="large" /></el-form-item>
        <el-form-item><el-input v-model="password" type="password" :placeholder="t('login.password')" size="large" show-password @keyup.enter="submit" /></el-form-item>
        <el-form-item>
          <el-button type="primary" style="width:100%" size="large" @click="submit" :loading="loading">
            {{ isRegister ? t('login.register') : t('login.login') }}
          </el-button>
        </el-form-item>
      </el-form>
      <el-button type="text" style="width:100%" @click="isRegister = !isRegister">
        {{ isRegister ? t('login.switchToLogin') : t('login.switchToRegister') }}
      </el-button>
      <el-alert v-if="error" :title="error" type="error" show-icon style="margin-top:16px" @close="error=''" />
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
const error = ref('')
const isRegister = ref(false)

async function submit() {
  if (!email.value || !password.value) return
  loading.value = true
  error.value = ''
  try {
    const fn = isRegister.value ? auth.register : auth.login
    const res = await fn(email.value, password.value)
    const d = res.data.data
    store.login(d.token, { id: d.user_id, email: d.email, role: d.role })
    const redirect = route.query.redirect as string
    router.push(redirect || '/')
  } catch (e: any) {
    error.value = e.response?.data?.message || t('login.requestFailed')
  } finally {
    loading.value = false
  }
}
</script>
