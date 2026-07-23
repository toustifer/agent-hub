<template>
  <MainLayout>
    <div class="mcp-page">
      <div class="mcp-head">
        <div>
          <h2>{{ t('mcp.title') }}</h2>
          <p class="sub">{{ t('mcp.subtitle') }}</p>
        </div>
      </div>

      <el-card shadow="never" class="ai-card">
        <div class="ai-row">
          <div>
            <div class="ai-title">{{ t('mcp.aiDoc') }}</div>
            <div class="ai-hint">{{ t('mcp.aiDocHint') }}</div>
            <code class="url">{{ mcpMdUrl }}</code>
          </div>
          <div class="ai-actions">
            <el-button type="primary" @click="copyUrl">{{ copied ? t('mcp.copied') : t('mcp.copyUrl') }}</el-button>
            <el-button @click="open(mcpMdUrl)">{{ t('mcp.openAiDoc') }}</el-button>
            <el-button text type="primary" @click="open(setupUrl)">{{ t('mcp.humanSetup') }}</el-button>
          </div>
        </div>
      </el-card>

      <el-row :gutter="16" style="margin-top:16px">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header><span>{{ t('mcp.step1') }}</span></template>
            <p class="hint">{{ t('mcp.step1hint') }}</p>
            <pre class="code"><button class="copy" @click="copy(remoteMcpJson)">{{ t('mcp.copyUrl') }}</button>{{ remoteMcpJson }}</pre>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header><span>{{ t('mcp.step2') }}</span></template>
            <p class="hint">{{ t('mcp.step2hint') }}</p>
            <pre class="code"><button class="copy" @click="copy(mcpEndpoint)">{{ t('mcp.copyUrl') }}</button>{{ mcpEndpoint }}</pre>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16" style="margin-top:16px">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header><span>{{ t('mcp.step3') }}</span></template>
            <p class="body">{{ t('mcp.step3body') }}</p>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header><span>{{ t('mcp.step4') }}</span></template>
            <p class="body">{{ t('mcp.step4body') }}</p>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never" style="margin-top:16px">
        <template #header><span>{{ t('mcp.authTitle') }}</span></template>
        <ul class="auth-list">
          <li>{{ t('mcp.authHuman') }}</li>
          <li>{{ t('mcp.authMachine') }}</li>
        </ul>
      </el-card>

      <el-collapse style="margin-top:16px">
        <el-collapse-item :title="t('mcp.legacyTitle')" name="legacy">
          <p class="hint">{{ t('mcp.legacyHint') }}</p>
          <pre class="code"><button class="copy" @click="copy(legacyMcpJson)">{{ t('mcp.copyUrl') }}</button>{{ legacyMcpJson }}</pre>
        </el-collapse-item>
      </el-collapse>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'
import { useI18n } from '@/i18n'

const { t } = useI18n()
const mcpMdUrl = 'https://hub.stifer.xyz/mcp.md'
const setupUrl = 'https://hub.stifer.xyz/setup'
const mcpEndpoint = 'https://hub.stifer.xyz/mcp'
const copied = ref(false)

const remoteMcpJson = `{
  "mcpServers": {
    "hub": {
      "type": "http",
      "url": "https://hub.stifer.xyz/mcp"
    }
  }
}`

const legacyMcpJson = `{
  "mcpServers": {
    "hub": {
      "command": "node",
      "args": ["ABS_PATH/mcp-server/index.js"],
      "env": {
        "HUB_API_URL": "https://hub.stifer.xyz"
      }
    }
  }
}`

function open(url: string) { window.open(url, '_blank') }

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('mcp.copied'))
  } catch {
    ElMessage.error('copy failed')
  }
}

async function copyUrl() {
  await copy(mcpMdUrl)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}
</script>

<style scoped>
.mcp-page { max-width: 1100px; }
.mcp-head h2 { margin: 0 0 6px; font-size: 22px; }
.sub { margin: 0; color: #909399; font-size: 14px; }
.ai-card { background: linear-gradient(135deg, #1a1a2e, #16213e); border: none; color: #e0e0e0; }
.ai-card :deep(.el-card__body) { padding: 18px 20px; }
.ai-row { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; flex-wrap: wrap; }
.ai-title { font-size: 16px; font-weight: 600; color: #fff; margin-bottom: 4px; }
.ai-hint { font-size: 13px; color: #a0a0b0; margin-bottom: 8px; }
.url { font-size: 13px; color: #7dd3fc; }
.ai-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.hint { margin: 0 0 8px; color: #909399; font-size: 13px; }
.body { margin: 0; color: #606266; font-size: 14px; line-height: 1.6; }
.code { position: relative; background: #0f172a; color: #e2e8f0; padding: 14px 16px; border-radius: 8px; font-size: 12px; overflow: auto; white-space: pre-wrap; }
.copy { position: absolute; top: 8px; right: 8px; font-size: 11px; cursor: pointer; background: #1e293b; color: #94a3b8; border: 1px solid #334155; border-radius: 4px; padding: 2px 8px; }
.auth-list { margin: 0; padding-left: 18px; color: #606266; line-height: 1.8; }
</style>
