<template>
  <MainLayout>
    <div v-if="team">
      <div style="display:flex;align-items:center;gap:16px;margin-bottom:20px">
        <div style="width:56px;height:56px;border-radius:12px;background:#409EFF;color:#fff;display:flex;align-items:center;justify-content:center;font-size:24px;font-weight:bold">{{ team.code[0].toUpperCase() }}</div>
        <div>
          <h2 style="margin:0">{{ team.name }}</h2>
          <span style="color:#909399">{{ team.code }} · {{ team.status }}</span>
        </div>
      </div>

      <el-alert v-if="errorMsg" :title="errorMsg" type="error" show-icon style="margin-bottom:12px" />
      <el-tabs v-model="tab">
        <!-- Overview tab -->
        <el-tab-pane :label="$t('team.overview')" name="overview">
          <el-row :gutter="16" style="margin-bottom:16px">
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#409EFF">{{ workers.length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.totalWorkers') }}</div></el-card></el-col>
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#67c23a">{{ workers.filter(w=>w.status==='online').length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.onlineWorkers') }}</div></el-card></el-col>
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#e6a23c">{{ locks.length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.activeLocks') }}</div></el-card></el-col>
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#909399">{{ dag.filter(t=>t.status==='pending'||t.status==='in_progress').length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.pendingTasks') }}</div></el-card></el-col>
          </el-row>
          <el-card shadow="never" style="margin-bottom:16px" v-if="team">
            <div style="font-size:14px;font-weight:bold;margin-bottom:8px">{{ team.name }}</div>
            <div style="color:#909399;font-size:13px">{{ team.description || $t('team.noDescription') }}</div>
            <div style="margin-top:8px"><el-tag size="small" :type="team.status==='active'?'success':'warning'">{{ team.status }}</el-tag></div>
          </el-card>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-card shadow="never">
                <template #header><span style="font-weight:bold">{{ $t('team.recentActivity') }}</span></template>
                <div v-if="events.length"><div v-for="e in events.slice(0,8)" :key="e.id" style="font-size:12px;padding:4px 0;border-bottom:1px solid #2a2a2a"><el-tag size="small" type="info" style="margin-right:6px">{{ e.event_type }}</el-tag><span style="color:#909399">{{ e.actor }} · {{ (e.created_at||'').slice(0,16) }}</span></div></div>
                <div v-else style="color:#909399;font-size:13px">{{ $t('team.noEvents') }}</div>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="never">
                <template #header><span style="font-weight:bold">{{ $t('team.workerList') }}</span></template>
                <div v-if="workers.length"><div v-for="w in workers.slice(0,8)" :key="w.worker_id" style="font-size:12px;padding:4px 0;border-bottom:1px solid #2a2a2a;cursor:pointer" @click="showWorker(w)"><span :style="{color:isStale(w)?'#f56c6c':'#e0e0e0'}">{{ w.worker_id }}</span><span style="color:#909399;margin-left:8px">{{ w.owner||'-' }}</span><el-tag size="small" :type="isStale(w)?'danger':w.status==='online'?'dark':'info'" style="margin-left:8px">{{ isStale(w)?'stale':w.status }}</el-tag></div></div>
                <div v-else style="color:#909399;font-size:13px">No workers</div>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane name="workers"><template #label>{{ $t('team.workers') }} <el-badge v-if="staleCount" :value="staleCount" type="danger" style="margin-left:4px" /></template>
          <el-empty v-if="!loading && !workers.length" description="No workers" />
          <el-table v-else :data="workers" class="dark-table" v-loading="loading" stripe @row-click="showWorker" highlight-current-row>
            <el-table-column prop="worker_id" :label="$t('team.workerId')" width="200" />
            <el-table-column prop="version" :label="$t('team.version')" width="90" />
            <el-table-column prop="owner" :label="$t('team.owner')" width="100" />
            <el-table-column prop="host" :label="$t('team.host')" width="130" />
            <el-table-column prop="status" :label="$t('team.status')" width="100">
              <template #default="{row}"><el-tag :type="isStale(row)?'danger':row.status==='online'?'dark':'info'">{{ isStale(row)?'stale':row.status }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="last_heartbeat_at" :label="$t('team.lastHeartbeat')" width="200" />
            <el-table-column label="" width="60"><template #default><span style="color:#909399;font-size:12px">›</span></template></el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="$t('team.dag')" name="dag">
          <el-empty v-if="!loading && !dag.length" :description="$t('team.noDag')" />
          <el-table v-else :data="dag" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="task_id" :label="$t('team.taskId')" width="100" />
            <el-table-column prop="title" :label="$t('team.taskTitle')" />
            <el-table-column prop="status" :label="$t('team.taskStatus')" width="120">
              <template #default="{r}"><el-tag :type="r.status==='completed'?'success':r.status==='in_progress'?'warning':'info'" size="small">{{ r.status }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="assigned_worker" :label="$t('team.taskWorker')" width="180" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="$t('team.locks')" name="locks">
          <el-table :data="locks" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="resource_key" :label="$t('team.resource')" width="300" />
            <el-table-column prop="holder_worker_id" :label="$t('team.holder')" width="200" />
            <el-table-column prop="acquired_at" :label="$t('team.acquired')" width="200" />
            <el-table-column prop="expires_at" :label="$t('team.expires')" width="200" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="$t('team.playbooks')" name="playbooks">
          <el-input v-model="searchQ" :placeholder="$t('team.search')" style="width:300px;margin-bottom:16px" @keyup.enter="loadPlaybooks" />
          <el-table :data="playbooks" v-loading="loading" stripe @row-click="showPb">
            <el-table-column prop="title" :label="$t('team.title')" width="250" />
            <el-table-column prop="category" :label="$t('team.category')" width="120"><template #default="{row}"><el-tag>{{ row.category }}</el-tag></template></el-table-column>
            <el-table-column prop="tags" :label="$t('team.tags')"><template #default="{row}">{{ (row.tags||[]).join(', ') }}</template></el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane name="events"><template #label>{{ $t('team.events') }} <span v-if="sseConnected" style="color:#67c23a;font-size:10px">● LIVE</span><span v-else style="color:#909399;font-size:10px"> ●</span></template>
          <el-timeline>
            <el-timeline-item v-for="ev in events" :key="ev.id" :timestamp="ev.created_at">
              <strong>{{ ev.event_type }}</strong> by {{ ev.actor }}
              <p style="color:#909399">{{ JSON.stringify(ev.payload) }}</p>
            </el-timeline-item>
          </el-timeline>
        </el-tab-pane>
      </el-tabs>

      <el-dialog v-model="pbVisible" :title="pbDetail?.title" width="700px" class="pb-dialog">
        <div class="pb-content" v-html="fmtContent(pbDetail?.content || '')" />
      </el-dialog>

      <el-drawer v-model="workerVisible" :title="$t('team.workerDetail')" size="450px">
        <template v-if="workerDetail">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item :label="$t('team.workerId')">{{ workerDetail.worker_id }}</el-descriptions-item>
            <el-descriptions-item :label="$t('team.owner')">{{ workerDetail.owner }}</el-descriptions-item>
            <el-descriptions-item :label="$t('team.status')"><el-tag :type="workerDetail.status==='online'?'dark':'info'">{{ workerDetail.status }}</el-tag></el-descriptions-item>
            <el-descriptions-item :label="$t('team.version')">{{ workerDetail.version }}</el-descriptions-item>
            <el-descriptions-item :label="$t('team.host')">{{ workerDetail.host }}</el-descriptions-item>
            <el-descriptions-item :label="$t('team.pid')">{{ workerDetail.pid }}</el-descriptions-item>
            <el-descriptions-item :label="$t('team.lastHeartbeat')">{{ workerDetail.last_heartbeat_at }}</el-descriptions-item>
          </el-descriptions>
          <el-button type="warning" size="small" @click="openPublishDialog" style="margin-top:12px">{{ $t('community.publish') }}</el-button>
          <template v-if="workerDetail.handbook">
            <el-divider>{{ $t('team.handbook') }}</el-divider>
            <div v-if="workerDetail.handbook.business_flow" style="font-size:13px;color:#909399;margin-bottom:12px">
              <strong>{{ $t('team.businessFlow') }}:</strong> {{ workerDetail.handbook.business_flow }}
            </div>
            <div v-if="workerDetail.handbook.code_map && workerDetail.handbook.code_map.length" style="margin-bottom:12px">
              <strong style="font-size:13px">{{ $t('team.codeMap') }}:</strong>
              <div v-for="f in workerDetail.handbook.code_map" :key="f.path" style="font-size:12px;color:#909399;margin:2px 0;padding-left:8px">
                <code style="color:#67c23a;font-size:11px">{{ f.path }}</code>
                <span v-if="f.purpose" style="margin-left:4px">— {{ f.purpose }}</span>
              </div>
            </div>
            <div v-if="workerDetail.handbook.danger_zones && workerDetail.handbook.danger_zones.length" style="margin-bottom:12px">
              <strong style="font-size:13px;color:#f56c6c">{{ $t('team.dangerZones') }}:</strong>
              <div v-for="d in workerDetail.handbook.danger_zones" :key="d.path" style="font-size:12px;margin:2px 0;padding-left:8px">
                <code style="color:#e6a23c;font-size:11px">{{ d.path }}</code>
                <span v-if="d.why"> — {{ d.why }}</span>
              </div>
            </div>
          </template>
          <el-divider>{{ $t('team.playbooksCap') }}</el-divider>
          <div v-if="workerPlaybooks.length">
            <el-card v-for="p in workerPlaybooks" :key="p.id" shadow="hover" style="margin-bottom:8px" @click="showPb(p)">
              <div style="font-weight:bold;font-size:14px">{{ p.title }}</div>
              <div style="display:flex;gap:6px;margin-top:4px">
                <el-tag size="small">{{ p.category }}</el-tag>
                <el-tag v-for="tag in (p.tags||[])" :key="tag" size="small" type="info">{{ tag }}</el-tag>
              </div>
            </el-card>
          </div>
          <div v-else style="color:#909399;font-size:13px">{{ $t('team.noPlaybooks') }}</div>
          <el-divider>{{ $t('team.recentEvents') }}</el-divider>
          <div v-if="workerEvents.length">
            <div v-for="e in workerEvents" :key="e.id" style="margin-bottom:8px;font-size:13px">
              <strong>{{ e.event_type }}</strong>
              <span style="color:#909399;margin-left:8px">{{ (e.created_at||'').slice(0,16) }}</span>
            </div>
          </div>
          <div v-else style="color:#909399;font-size:13px">{{ $t('team.noEvents') }}</div>
        </template>
      </el-drawer>

      <!-- Publish Dialog -->
      <el-dialog v-model="publishVisible" :title="$t('community.publishWorker')" width="600px">
        <el-form label-position="top">
          <el-form-item label="Title">
            <el-input v-model="publishForm.title" />
          </el-form-item>
          <el-form-item label="Description">
            <el-input v-model="publishForm.description" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item :label="$t('community.domainFilter')">
            <el-select v-model="publishForm.domain" style="width:100%">
              <el-option label="frontend" value="frontend" />
              <el-option label="backend" value="backend" />
              <el-option label="testing" value="testing" />
              <el-option label="ble" value="ble" />
              <el-option label="devops" value="devops" />
              <el-option label="ai" value="ai" />
              <el-option label="hardware" value="hardware" />
              <el-option label="other" value="other" />
            </el-select>
          </el-form-item>
          <el-form-item label="Scope">
            <el-input v-model="publishForm.scope" />
          </el-form-item>
          <el-form-item label="Tags (comma-separated)">
            <el-input v-model="publishForm.tags" placeholder="e.g. vue, typescript, api" />
          </el-form-item>
          <el-form-item>
            <el-checkbox v-model="publishForm.deidentify">{{ $t('community.deidentify') }}</el-checkbox>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="publishVisible = false">{{ $t('dash.cancel') }}</el-button>
          <el-button type="primary" :loading="publishing" @click="doPublish">{{ $t('community.publish') }}</el-button>
        </template>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { getBusinesses, getWorkers, getLocks, getEvents, searchPlaybooks, getDAG, publishCommunityWorker } from '@/api/hub'
import { useI18n } from '@/i18n'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'

const route = useRoute()
const { t: $t } = useI18n()
const tab = ref('overview')
const team = ref<any>(null)
const workers = ref<any[]>([])
const locks = ref<any[]>([])
const playbooks = ref<any[]>([])
const events = ref<any[]>([])
const dag = ref<any[]>([])
const searchQ = ref('')
const pbDetail = ref<any>(null)
const pbVisible = ref(false)
const workerDetail = ref<any>(null)
const workerVisible = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const sseConnected = ref(false)
const liveCount = ref(0)
let timer: number
let eventSource: EventSource | null = null

function isStale(row: any) { if (!row.last_heartbeat_at) return true; return Date.now() - new Date(row.last_heartbeat_at).getTime() > 90000 }
const staleCount = computed(() => workers.value.filter(isStale).length)

function connectSSE(code: string) {
  if (eventSource) eventSource.close()
  const token = localStorage.getItem('token') || ''
  eventSource = new EventSource(`https://hub.stifer.xyz/v1/hub/events/stream?business=${code}&token=${token}`)
  eventSource.onopen = () => { sseConnected.value = true }
  eventSource.onmessage = (e) => { try { const ev = JSON.parse(e.data); events.value.unshift(ev); if (events.value.length > 100) events.value.pop(); liveCount.value++ } catch {} }
  eventSource.onerror = () => { sseConnected.value = false; eventSource?.close() }
}
function disconnectSSE() { eventSource?.close(); eventSource = null; sseConnected.value = false }

async function load() {
  const code = route.params.code as string
  try { const bizRes = await getBusinesses(); team.value = (bizRes.data?.data || []).find((b: any) => b.code === code) } catch (e) { console.error(e) }
  loadTab()
}
async function loadTab() {
  const code = route.params.code as string
  loading.value = true; errorMsg.value = ''
  try {
    if (tab.value === 'overview' || tab.value === 'workers') { const r = await getWorkers({ business: code }); workers.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'locks') { const r = await getLocks({ business: code }); locks.value = r.data?.data || [] }
    if (tab.value === 'playbooks') { const r = await searchPlaybooks({ q: searchQ.value || '', business: code }); playbooks.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'events') { const r = await getEvents({ business: code, limit: 20 }); events.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'dag') { const r = await getDAG(code); dag.value = r.data?.data || [] }
  } catch (e: any) { errorMsg.value = e?.message || 'Failed to load' }
  loading.value = false
}
function loadPlaybooks() { tab.value = 'playbooks'; loadTab() }
function showPb(row: any) { pbDetail.value = row; pbVisible.value = true }
function fmtContent(text: string) { return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\n/g, '<br>') }
const workerPlaybooks = ref<any[]>([])
const workerEvents = ref<any[]>([])

// Publish to community
const publishVisible = ref(false)
const publishing = ref(false)
const publishForm = ref({ title: '', description: '', domain: 'other', scope: '', tags: '', deidentify: true })

function openPublishDialog() {
  if (!workerDetail.value) return
  publishForm.value = {
    title: workerDetail.value.worker_id || '',
    description: '',
    domain: 'other',
    scope: workerDetail.value.scope || '',
    tags: '',
    deidentify: true,
  }
  publishVisible.value = true
}

async function doPublish() {
  publishing.value = true
  try {
    const tags = publishForm.value.tags ? publishForm.value.tags.split(',').map((t: string) => t.trim()).filter(Boolean) : []
    await publishCommunityWorker({
      worker_id: workerDetail.value.worker_id,
      business_code: route.params.code as string,
      title: publishForm.value.title,
      description: publishForm.value.description,
      domain: publishForm.value.domain,
      scope: publishForm.value.scope,
      tags,
      deidentify: publishForm.value.deidentify,
    })
    ElMessage.success($t('community.publishSuccess'))
    publishVisible.value = false
  } catch (e: any) {
    ElMessage.error(e?.message || 'Publish failed')
  }
  publishing.value = false
}

async function showWorker(row: any) {
  workerDetail.value = row; workerVisible.value = true; workerPlaybooks.value = []; workerEvents.value = []
  try {
    const [pRes, eRes] = await Promise.all([searchPlaybooks({ q: '', business: route.params.code as string, limit: 50 }), getEvents({ business: route.params.code as string, limit: 50 })])
    workerPlaybooks.value = (pRes.data?.data || []).filter((p: any) => p.created_by_worker_id === row.worker_id)
    workerEvents.value = (eRes.data?.data || []).filter((e: any) => e.actor === row.worker_id)
  } catch (e: any) { errorMsg.value = e?.message || 'Failed to load worker info' }
}

onMounted(() => { load(); timer = window.setInterval(loadTab, 10000); connectSSE(route.params.code as string) })
onUnmounted(() => { clearInterval(timer); disconnectSSE() })
watch(tab, loadTab)
</script>

<style>
.dark-table { --el-table-bg-color: #1a1a1a; --el-table-tr-bg-color: #1a1a1a; --el-table-header-bg-color: #1a1a1a; --el-table-border-color: #2a2a2a; --el-table-text-color: #ccc; --el-table-row-hover-bg-color: #2a2a2a; }
.dark-table .el-table__row--striped { --el-table-tr-bg-color: #222 !important; }
.pb-content { font-size:14px; line-height:1.8; color:#e0e0e0; }
.pb-content code { background:#2a2a2a; color:#67c23a; padding:2px 6px; border-radius:4px; font-size:13px; font-family:'Cascadia Code', 'Fira Code', monospace; }
.pb-dialog .el-dialog__body { padding-top:8px; }
</style>
