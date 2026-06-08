<template>
  <MainLayout>
    <div>
      <el-alert v-if="errorMsg" :title="errorMsg" type="error" show-icon style="margin-bottom:12px" />

      <div v-if="worker" v-loading="loading">
        <!-- Header -->
        <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px">
          <el-button size="small" @click="goBack">{{ $t('community.back') }}</el-button>
          <div style="flex:1">
            <h2 style="margin:0;display:flex;align-items:center;gap:12px">
              {{ worker.title || worker.worker_id }}
              <el-tag :color="domainColor(worker.domain)" size="small" effect="dark" style="border:none">
                {{ worker.domain || 'other' }}
              </el-tag>
            </h2>
            <div style="color:#909399;font-size:13px;margin-top:4px">
              {{ $t('community.publisher') }}: {{ worker.publisher_name || worker.publisher_id || '-' }}
              · <el-icon style="vertical-align:middle"><Download /></el-icon> {{ worker.install_count || 0 }} {{ $t('community.installs') }}
            </div>
          </div>
          <el-button type="primary" @click="openInstallDialog">{{ $t('community.install') }}</el-button>
        </div>

        <!-- Description -->
        <el-card shadow="never" style="margin-bottom:16px">
          <p style="font-size:14px;color:#909399;margin:0;white-space:pre-wrap">{{ worker.description || $t('team.noDescription') }}</p>
          <div v-if="worker.tags && worker.tags.length" style="margin-top:8px;display:flex;gap:4px;flex-wrap:wrap">
            <el-tag v-for="tag in worker.tags" :key="tag" size="small" type="info">{{ tag }}</el-tag>
          </div>
        </el-card>

        <!-- Tabs -->
        <el-tabs v-model="tab">
          <!-- Handbook Tab -->
          <el-tab-pane :label="$t('team.handbook')" name="handbook">
            <el-card v-if="worker.handbook" shadow="never">
              <div v-if="worker.handbook.business_flow" style="margin-bottom:16px">
                <strong style="font-size:14px">{{ $t('team.businessFlow') }}</strong>
                <p style="font-size:13px;color:#909399;margin:8px 0 0;white-space:pre-wrap">{{ worker.handbook.business_flow }}</p>
              </div>
              <div v-if="worker.handbook.code_map && worker.handbook.code_map.length" style="margin-bottom:16px">
                <strong style="font-size:14px">{{ $t('team.codeMap') }}</strong>
                <div v-for="f in worker.handbook.code_map" :key="f.path" style="font-size:12px;color:#909399;margin:4px 0;padding-left:8px">
                  <code style="color:#67c23a;font-size:11px">{{ f.path }}</code>
                  <span v-if="f.purpose" style="margin-left:4px">— {{ f.purpose }}</span>
                </div>
              </div>
              <div v-if="worker.handbook.danger_zones && worker.handbook.danger_zones.length">
                <strong style="font-size:14px;color:#f56c6c">{{ $t('team.dangerZones') }}</strong>
                <div v-for="d in worker.handbook.danger_zones" :key="d.path" style="font-size:12px;margin:4px 0;padding-left:8px">
                  <code style="color:#e6a23c;font-size:11px">{{ d.path }}</code>
                  <span v-if="d.why"> — {{ d.why }}</span>
                </div>
              </div>
            </el-card>
            <el-empty v-else description="No handbook data" />
          </el-tab-pane>

          <!-- Playbooks Tab -->
          <el-tab-pane :label="$t('team.playbooksCap')" name="playbooks">
            <el-empty v-if="!playbooks.length" :description="$t('team.noPlaybooks')" />
            <div v-else>
              <el-card v-for="p in playbooks" :key="p.id" shadow="never" class="pb-card" style="margin-bottom:12px">
                <div style="font-weight:bold;font-size:14px">{{ p.title }}</div>
                <div style="display:flex;gap:6px;margin-top:6px">
                  <el-tag size="small">{{ p.category }}</el-tag>
                  <el-tag v-for="tag in (p.tags||[])" :key="tag" size="small" type="info">{{ tag }}</el-tag>
                </div>
                <div v-if="p.content" class="pb-content" style="font-size:13px;color:#909399;margin-top:8px;line-height:1.6" v-html="fmtContent(p.content)" />
              </el-card>
            </div>
          </el-tab-pane>

          <!-- Reviews Tab -->
          <el-tab-pane name="reviews"><template #label>{{ $t('community.reviews') }} <el-badge v-if="reviews.length" :value="reviews.length" type="primary" style="margin-left:4px" /></template>
            <el-empty v-if="!reviews.length" :description="$t('community.noReviews')" />
            <div v-else>
              <el-card v-for="r in reviews" :key="r.id" shadow="never" style="margin-bottom:12px">
                <div style="display:flex;align-items:center;gap:12px">
                  <el-rate v-model="r.rating" disabled show-score text-color="#e6a23c" />
                  <span style="color:#909399;font-size:12px">{{ r.user_name || r.user_id || '-' }}</span>
                  <span style="color:#909399;font-size:12px">{{ (r.created_at || '').slice(0, 16) }}</span>
                </div>
                <p v-if="r.comment" style="font-size:13px;color:#909399;margin:8px 0 0 0">{{ r.comment }}</p>
              </el-card>
            </div>
          </el-tab-pane>
        </el-tabs>

        <!-- Install Dialog -->
        <el-dialog v-model="installVisible" :title="$t('community.install')" width="450px">
          <el-form label-position="top">
            <el-form-item :label="$t('community.selectProject')">
              <el-select v-model="selectedBusiness" style="width:100%" placeholder="Select business...">
                <el-option v-for="b in businesses" :key="b.code" :label="b.code + ' - ' + (b.name || '')" :value="b.code" />
              </el-select>
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="installVisible = false">{{ $t('dash.cancel') }}</el-button>
            <el-button type="primary" :disabled="!selectedBusiness" :loading="installing" @click="doInstall">
              {{ $t('community.install') }}
            </el-button>
          </template>
        </el-dialog>
      </div>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getCommunityWorker, getCommunityWorkerReviews, installCommunityWorker } from '@/api/hub'
import { me } from '@/api/hub'
import { useI18n } from '@/i18n'
import { Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'

const route = useRoute()
const router = useRouter()
const { t: $t } = useI18n()

const worker = ref<any>(null)
const playbooks = ref<any[]>([])
const reviews = ref<any[]>([])
const tab = ref('handbook')
const loading = ref(false)
const errorMsg = ref('')

// Install dialog
const installVisible = ref(false)
const businesses = ref<any[]>([])
const selectedBusiness = ref('')
const installing = ref(false)

function domainColor(domain: string): string {
  const colors: Record<string, string> = {
    frontend: '#409EFF', backend: '#67C23A', testing: '#E6A23C',
    ble: '#F56C6C', devops: '#909399', ai: '#9B59B6',
    hardware: '#1ABC9C', other: '#95A5A6'
  }
  return colors[domain] || '#95A5A6'
}

function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/community')
  }
}

function fmtContent(text: string) {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\n/g, '<br>')
}

async function load() {
  const id = Number(route.params.id)
  if (!id) { errorMsg.value = 'Invalid worker ID'; return }

  loading.value = true
  errorMsg.value = ''
  try {
    const [wRes, rRes] = await Promise.all([
      getCommunityWorker(id),
      getCommunityWorkerReviews(id),
    ])
    worker.value = wRes.data?.data
    reviews.value = rRes.data?.data || []
    // Parse playbooks from worker's JSONB field
    if (worker.value?.playbooks) {
      const pb = worker.value.playbooks
      playbooks.value = Array.isArray(pb) ? pb : Object.values(pb)
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Failed to load worker'
  }
  loading.value = false
}

async function openInstallDialog() {
  installVisible.value = true
  selectedBusiness.value = ''
  try {
    const res = await me.getBusinesses()
    businesses.value = res.data?.data || []
  } catch { businesses.value = [] }
}

async function doInstall() {
  if (!selectedBusiness.value) return
  installing.value = true
  try {
    const id = Number(route.params.id)
    await installCommunityWorker(id, selectedBusiness.value)
    ElMessage.success($t('community.installSuccess'))
    installVisible.value = false
  } catch (e: any) {
    ElMessage.error(e?.message || 'Install failed')
  }
  installing.value = false
}

onMounted(() => load())
</script>

<style>
.pb-content code {
  background: #2a2a2a;
  color: #67c23a;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'Cascadia Code', 'Fira Code', monospace;
}

/* Dark mode overrides */
html.dark .pb-card {
  background: #1e1e1e !important;
  border-color: #333 !important;
}
html.dark .worker-title {
  color: #e0e0e0;
}
html.dark .el-empty__description {
  color: #909399;
}
html.dark .el-form-item__label {
  color: #ccc !important;
}
</style>
