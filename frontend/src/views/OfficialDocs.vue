<template>
  <MainLayout>
    <div class="docs-page">
      <div class="docs-head">
        <div>
          <h2>{{ t('docs.title') }}</h2>
          <p class="sub">{{ t('docs.subtitle') }}</p>
        </div>
      </div>

      <el-card shadow="never" class="ai-card">
        <div class="ai-row">
          <div>
            <div class="ai-title">{{ t('docs.aiPaste') }}</div>
            <div class="ai-hint">{{ t('docs.aiPasteHint') }}</div>
            <code class="url">{{ agentSetupUrl }}</code>
          </div>
          <div class="ai-actions">
            <el-button type="primary" @click="copy(agentSetupUrl)">{{ t('docs.copy') }}</el-button>
            <el-button @click="open(agentSetupUrl)">{{ t('docs.open') }}</el-button>
          </div>
        </div>
      </el-card>

      <el-alert type="info" show-icon :closable="false" style="margin-top:16px" :title="t('docs.sceneTitle')">
        <div>{{ t('docs.scene1') }}</div>
        <div style="margin-top:6px">{{ t('docs.scene2') }}</div>
        <div style="margin-top:6px;color:#909399">{{ t('docs.sceneBoth') }}</div>
      </el-alert>

      <el-tabs v-model="tab" style="margin-top:16px">
        <!-- Codex CLI -->
        <el-tab-pane :label="t('docs.tabCodex')" name="codex">
          <el-card shadow="never" class="ai-card" style="margin-bottom:16px">
            <div class="ai-row">
              <div>
                <div class="ai-title">{{ t('docs.codexAiDoc') }}</div>
                <div class="ai-hint">{{ t('docs.codexAiDocHint') }}</div>
                <code class="url">{{ codexSetupUrl }}</code>
              </div>
              <div class="ai-actions">
                <el-button type="primary" @click="copy(codexSetupUrl)">{{ t('docs.copy') }}</el-button>
                <el-button @click="open(codexSetupUrl)">{{ t('docs.open') }}</el-button>
              </div>
            </div>
          </el-card>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.codexStep1') }}</span></template>
                <p class="hint">{{ t('docs.codexStep1hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(codexAddCmd)">{{ t('docs.copy') }}</button>{{ codexAddCmd }}</pre>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.codexStep2') }}</span></template>
                <p class="hint">{{ t('docs.codexStep2hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(codexLoginCmd)">{{ t('docs.copy') }}</button>{{ codexLoginCmd }}</pre>
              </el-card>
            </el-col>
          </el-row>
          <el-row :gutter="16" style="margin-top:16px">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.codexStep3') }}</span></template>
                <p class="body">{{ t('docs.codexStep3body') }}</p>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.codexStep4') }}</span></template>
                <p class="body">{{ t('docs.codexStep4body') }}</p>
              </el-card>
            </el-col>
          </el-row>
          <el-card shadow="never" style="margin-top:16px">
            <template #header><span>{{ t('docs.codexAuthTitle') }}</span></template>
            <p class="body">{{ t('docs.codexAuthBearer') }}</p>
            <p class="hint" style="margin-top:12px">{{ t('docs.codexTomlTitle') }}</p>
            <pre class="code"><button class="copy" type="button" @click="copy(codexToml)">{{ t('docs.copy') }}</button>{{ codexToml }}</pre>
          </el-card>
        </el-tab-pane>

        <!-- Scenario 1: agentflow first (personal local must) -->
        <el-tab-pane :label="t('docs.tabAgentflow')" name="agentflow">
          <el-card shadow="never" class="ai-card" style="margin-bottom:16px">
            <div class="ai-row">
              <div>
                <div class="ai-title">{{ t('docs.afAiDoc') }}</div>
                <div class="ai-hint">{{ t('docs.afAiDocHint') }}</div>
                <code class="url">{{ agentflowDocUrl }}</code>
              </div>
              <div class="ai-actions">
                <el-button type="primary" @click="copy(agentflowDocUrl)">{{ t('docs.copy') }}</el-button>
                <el-button @click="open(agentflowDocUrl)">{{ t('docs.open') }}</el-button>
              </div>
            </div>
          </el-card>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.afStep1') }}</span></template>
                <p class="hint">{{ t('docs.afStep1hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(agentflowOnlyJson)">{{ t('docs.copy') }}</button>{{ agentflowOnlyJson }}</pre>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.afStep2') }}</span></template>
                <p class="hint">{{ t('docs.afStep2hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(hubClientJson)">{{ t('docs.copy') }}</button>{{ hubClientJson }}</pre>
              </el-card>
            </el-col>
          </el-row>
          <el-row :gutter="16" style="margin-top:16px">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.afStep3') }}</span></template>
                <p class="body">{{ t('docs.afStep3body') }}</p>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.afStep4') }}</span></template>
                <p class="body">{{ t('docs.afStep4body') }}</p>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>

        <!-- Scenario 2: Hub team only -->
        <el-tab-pane :label="t('docs.tabHub')" name="hub">
          <el-row :gutter="16">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.hubStep1') }}</span></template>
                <p class="hint">{{ t('docs.hubStep1hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(remoteMcpJson)">{{ t('docs.copy') }}</button>{{ remoteMcpJson }}</pre>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.hubStep2') }}</span></template>
                <p class="hint">{{ t('docs.hubStep2hint') }}</p>
                <pre class="code"><button class="copy" type="button" @click="copy(mcpEndpoint)">{{ t('docs.copy') }}</button>{{ mcpEndpoint }}</pre>
              </el-card>
            </el-col>
          </el-row>
          <el-row :gutter="16" style="margin-top:16px">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.hubStep3') }}</span></template>
                <p class="body">{{ t('docs.hubStep3body') }}</p>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header><span>{{ t('docs.hubStep4') }}</span></template>
                <p class="body">{{ t('docs.hubStep4body') }}</p>
              </el-card>
            </el-col>
          </el-row>
          <el-card shadow="never" style="margin-top:16px">
            <template #header><span>{{ t('docs.hubAuthTitle') }}</span></template>
            <ul class="auth-list">
              <li>{{ t('docs.hubAuthHuman') }}</li>
              <li>{{ t('docs.hubAuthMachine') }}</li>
            </ul>
          </el-card>
          <el-collapse style="margin-top:16px">
            <el-collapse-item :title="t('docs.optionalBothTitle')" name="both">
              <p class="hint">{{ t('docs.optionalBothHint') }}</p>
              <pre class="code"><button class="copy" type="button" @click="copy(bothMcpJson)">{{ t('docs.copy') }}</button>{{ bothMcpJson }}</pre>
            </el-collapse-item>
            <el-collapse-item :title="t('docs.hubLegacyTitle')" name="legacy">
              <p class="hint">{{ t('docs.hubLegacyHint') }}</p>
              <pre class="code"><button class="copy" type="button" @click="copy(legacyMcpJson)">{{ t('docs.copy') }}</button>{{ legacyMcpJson }}</pre>
            </el-collapse-item>
          </el-collapse>
        </el-tab-pane>
      </el-tabs>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'
import { useI18n } from '@/i18n'

const { t } = useI18n()
const tab = ref('agentflow')
const agentSetupUrl = 'https://hub.stifer.xyz/agent-setup.md'
const agentflowDocUrl = 'https://hub.stifer.xyz/agentflow-setup.md'
const codexSetupUrl = 'https://hub.stifer.xyz/codex-setup.md'
const mcpEndpoint = 'https://hub.stifer.xyz/mcp'

const codexAddCmd = 'codex mcp add hub --url https://hub.stifer.xyz/mcp'
const codexLoginCmd = 'codex mcp login hub'
const codexToml = `[mcp_servers.hub]
url = "https://hub.stifer.xyz/mcp"
# OAuth: codex mcp login hub
# Bearer fallback:
# bearer_token_env_var = "HUB_TOKEN"`

const agentflowOnlyJson = `{
  "mcpServers": {
    "agentflow": {
      "command": "C:\Users\YOU\.claude\skills\agentflow\bin\agentflow.exe",
      "args": ["stdio"],
      "type": "stdio"
    }
  }
}`

const remoteMcpJson = `{
  "mcpServers": {
    "hub": {
      "type": "http",
      "url": "https://hub.stifer.xyz/mcp"
    }
  }
}`

const bothMcpJson = `{
  "mcpServers": {
    "agentflow": {
      "command": "C:\Users\YOU\.claude\skills\agentflow\bin\agentflow.exe",
      "args": ["stdio"],
      "type": "stdio"
    },
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

const hubClientJson = `{
  "hub_url": "https://hub.stifer.xyz",
  "token": "<Hub JWT after login/OAuth>",
  "business_code": "<4-char team code from Dashboard>",
  "email": "<your@email>"
}`

function open(url: string) { window.open(url, '_blank') }

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('docs.copied'))
  } catch {
    ElMessage.error('copy failed')
  }
}
</script>

<style scoped>
.docs-page { max-width: 1100px; }
.docs-head h2 { margin: 0 0 6px; font-size: 22px; }
.sub { margin: 0; color: #909399; font-size: 14px; }
.ai-card { background: linear-gradient(135deg, #1a1a2e, #16213e); border: none; color: #e0e0e0; }
.ai-card :deep(.el-card__body) { padding: 18px 20px; }
.ai-row { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; flex-wrap: wrap; }
.ai-title { font-size: 16px; font-weight: 600; color: #fff; margin-bottom: 4px; }
.ai-hint { font-size: 13px; color: #a0a0b0; margin-bottom: 8px; }
.url { font-size: 13px; color: #7dd3fc; word-break: break-all; }
.ai-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.hint { margin: 0 0 8px; color: #909399; font-size: 13px; }
.body { margin: 0; color: #606266; font-size: 14px; line-height: 1.6; }
.code { position: relative; background: #0f172a; color: #e2e8f0; padding: 14px 16px; border-radius: 8px; font-size: 12px; overflow: auto; white-space: pre-wrap; }
.copy { position: absolute; top: 8px; right: 8px; font-size: 11px; cursor: pointer; background: #1e293b; color: #94a3b8; border: 1px solid #334155; border-radius: 4px; padding: 2px 8px; }
.auth-list { margin: 0; padding-left: 18px; color: #606266; line-height: 1.8; }
</style>
