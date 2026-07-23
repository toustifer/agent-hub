<template>
  <div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5">
    <el-card style="width:460px;text-align:center">
      <div v-if="!tokenParam">
        <el-result icon="warning" title="Missing invite token" sub-title="Open the full invite_url you received." />
      </div>
      <div v-else-if="needLogin">
        <h2>Accept invitation</h2>
        <p style="color:#606266;margin:16px 0">Log in with the invited email, then accept.</p>
        <el-button type="primary" size="large" style="width:100%" @click="goLogin">Log in</el-button>
      </div>
      <div v-else-if="done">
        <el-result icon="success" title="Joined" sub-title="You are now a member of the team." />
        <el-button type="primary" @click="$router.push('/app')">Go to dashboard</el-button>
      </div>
      <div v-else>
        <h2>Accept team invitation</h2>
        <p style="color:#606266;margin:16px 0">Confirm with your logged-in account (must match invite email).</p>
        <el-alert v-if="error" :title="error" type="error" show-icon style="margin-bottom:12px" />
        <el-button type="primary" size="large" style="width:100%" :loading="loading" @click="accept">
          Accept invite
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { acceptInvite } from '@/api/hub'

const route = useRoute()
const router = useRouter()
const tokenParam = ref((route.query.token as string) || '')
const needLogin = ref(false)
const loading = ref(false)
const done = ref(false)
const error = ref('')

onMounted(() => {
  if (!localStorage.getItem('token')) needLogin.value = true
})

function goLogin() {
  const ret = encodeURIComponent('/invite/accept?token=' + encodeURIComponent(tokenParam.value))
  router.push('/login?redirect=' + ret)
}

async function accept() {
  if (!tokenParam.value) return
  loading.value = true
  error.value = ''
  try {
    await acceptInvite(tokenParam.value)
    done.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || 'Accept failed'
  }
  loading.value = false
}
</script>
