<template>
  <MainLayout>
    <div v-if="team">
      <div style="display:flex;align-items:center;gap:16px;margin-bottom:20px">
        <div style="width:56px;height:56px;border-radius:12px;background:#409EFF;color:#fff;display:flex;align-items:center;justify-content:center;font-size:24px;font-weight:bold">{{ (team.name || team.code || '?')[0] }}</div>
        <div style="flex:1">
          <h2 style="margin:0">{{ team.name }}</h2>
          <span style="color:#909399">{{ $t('dash.codeLabel') }}: <code>{{ team.code }}</code> · {{ team.status }}</span>
        </div>
        <el-button size="small" @click="openRename">{{ $t('team.rename') }}</el-button>
      </div>

      <el-alert v-if="errorMsg" :title="errorMsg" type="error" show-icon style="margin-bottom:12px" />
      <el-tabs v-model="tab">
        <!-- Overview tab -->
        <el-tab-pane :label="$t('team.overview')" name="overview">
          <el-row :gutter="16" style="margin-bottom:16px">
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#409EFF">{{ workerTemplates.length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.totalWorkers') }}</div><div style="color:#606266;font-size:11px;margin-top:4px">模板工种</div></el-card></el-col>
            <el-col :span="6"><el-card shadow="never" style="text-align:center"><div style="font-size:24px;font-weight:bold;color:#67c23a">{{ workers.filter(w=>w.status==='online'&&!isStale(w)).length }}</div><div style="color:#909399;font-size:12px">{{ $t('team.onlineWorkers') }}</div><div style="color:#606266;font-size:11px;margin-top:4px">心跳</div></el-card></el-col>
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
                <div v-if="events.length"><div v-for="e in events.slice(0,8)" :key="e.id" style="font-size:12px;padding:4px 0;border-bottom:1px solid #2a2a2a"><el-tag size="small" type="info" style="margin-right:6px">{{ e.event_type }}</el-tag><span style="color:#909399">{{ eventWho(e) }} · {{ (e.created_at||'').slice(0,16) }}</span></div></div>
                <div v-else style="color:#909399;font-size:13px">{{ $t('team.noEvents') }}</div>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="never">
                <template #header><span style="font-weight:bold">团队已定义 Worker（模板）</span></template>
                <div v-if="workerTemplates.length"><div v-for="w in workerTemplates.slice(0,10)" :key="w.worker_id" style="font-size:12px;padding:4px 0;border-bottom:1px solid #2a2a2a"><span style="color:#e0e0e0">{{ w.worker_id }}</span><span style="color:#909399;margin-left:8px">{{ w.display_name||'' }}</span></div>
                  <div v-if="workerTemplates.length>10" style="color:#909399;font-size:11px;margin-top:6px">共 {{ workerTemplates.length }} 个 · 见 Worker templates Tab</div>
                </div>
                <div v-else style="color:#909399;font-size:13px">暂无团队模板</div>
                <div v-if="workers.length" style="margin-top:10px;padding-top:8px;border-top:1px solid #333;font-size:11px;color:#909399">在线进程 {{ workers.filter(w=>!isStale(w)).length }} / 已注册 {{ workers.length }}</div>
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
            <el-table-column prop="assigned_worker" :label="$t('team.taskWorker')" width="140" />
            <el-table-column label="Assignee" width="180">
              <template #default="{ row }">
                <span style="font-size:12px">{{ row.assignee_email || row.last_actor_email || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="Last by" width="160">
              <template #default="{ row }">
                <span style="font-size:12px;color:#909399">{{ row.last_actor_email || '—' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane name="requirements" label="需求">
          <RequirementsTab :business-code="teamCode" />
        </el-tab-pane>
        <el-tab-pane name="board" label="看板">
          <KanbanBoard :business-code="teamCode" />
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
              <strong>{{ ev.event_type }}</strong>
              <span style="color:#e0e0e0"> · {{ eventWho(ev) }}</span>
              <span v-if="ev.actor_role" style="color:#909399;font-size:12px"> [{{ ev.actor_role }}]</span>
              <span v-if="ev.actor_worker_id" style="color:#909399;font-size:12px"> {{ ev.actor_worker_id }}</span>
              <p style="color:#909399">{{ JSON.stringify(ev.payload) }}</p>
            </el-timeline-item>
          </el-timeline>
        </el-tab-pane>
        
        <el-tab-pane name="docs" label="Docs">
          <el-empty v-if="!loading && !teamDocs.length" description="No team docs" />
          <el-table v-else :data="teamDocs" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="doc_key" label="Key" width="160" />
            <el-table-column prop="title" label="Title" />
            <el-table-column prop="category" label="Category" width="120" />
            <el-table-column prop="updated_by_email" label="Updated by" width="200" />
            <el-table-column prop="updated_at" label="Updated" width="180" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane name="templates" label="Worker templates">
          <el-empty v-if="!loading && !workerTemplates.length" description="No templates" />
          <el-table v-else :data="workerTemplates" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="worker_id" label="Worker" width="180" />
            <el-table-column prop="display_name" label="Name" />
            <el-table-column prop="published_by_email" label="Published by" width="200" />
            <el-table-column prop="visibility" label="Visibility" width="100" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="approvals"><template #label>Approvals <el-badge v-if="linkRequests.length" :value="linkRequests.length" type="warning" style="margin-left:4px" /></template>
          <el-empty v-if="!loading && !linkRequests.length" description="No pending requests" />
          <el-table v-else :data="linkRequests" class="dark-table" v-loading="loading" stripe>
            <el-table-column label="User" width="220">
              <template #default="{row}">
                <div style="font-weight:bold">{{ row.name || row.email }}</div>
                <div style="font-size:11px;color:#909399">{{ row.email }}</div>
              </template>
            </el-table-column>
            <el-table-column label="Device" width="200">
              <template #default="{row}"><span style="font-size:12px;color:#909399">{{ row.device_info || '-' }}</span></template>
            </el-table-column>
            <el-table-column prop="created_at" label="Requested" width="180" />
            <el-table-column label="Actions" width="200">
              <template #default="{row}">
                <el-button type="success" size="small" @click="reviewRequest(row.id, 'approve')">Approve</el-button>
                <el-button type="danger" size="small" @click="reviewRequest(row.id, 'reject')">Reject</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="branches" label="Branches">
          <div style="display:flex;gap:12px;margin-bottom:16px;align-items:center">
            <el-button type="primary" size="small" :loading="refreshingBranches" @click="doRefreshBranches">Refresh from GitHub / origin</el-button>
            <span style="color:#909399;font-size:12px">tip authority: github_api / ls_remote over report · stale = binding head ≠ tip</span>
          </div>
          <el-empty v-if="!loading && !branches.length" description="No branches reported yet" />
          <el-table v-else :data="branchRows" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="name" label="Branch" min-width="160">
              <template #default="{row}">
                <span>{{ row.name }}</span>
                <el-tag v-if="row.is_default" size="small" type="success" style="margin-left:6px">default</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Tip" width="110">
              <template #default="{row}"><code style="color:#67c23a">{{ (row.tip_sha||'').slice(0,8) || '--------' }}</code></template>
            </el-table-column>
            <el-table-column prop="source" label="Source" width="110">
              <template #default="{row}"><el-tag size="small" type="info">{{ row.source || '-' }}</el-tag></template>
            </el-table-column>
            <el-table-column label="PR" width="120">
              <template #default="{row}">
                <a v-if="row.pr_url" :href="row.pr_url" target="_blank" style="color:#409EFF">#{{ row.pr_number }}</a>
                <span v-else style="color:#909399">-</span>
              </template>
            </el-table-column>
            <el-table-column label="Bindings" min-width="220">
              <template #default="{row}">
                <template v-if="row.binds && row.binds.length">
                  <el-tag v-for="b in row.binds" :key="b.bind_type+':'+b.bind_id" size="small" :type="b.status==='stale'?'danger':'warning'" style="margin:2px">{{ b.bind_type }}:{{ b.bind_id }}{{ b.status==='stale'?' !':'' }}</el-tag>
                </template>
                <span v-else style="color:#909399">—</span>
              </template>
            </el-table-column>
            <el-table-column prop="last_seen_at" label="Seen" width="170" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="members" label="Members">
          <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap">
            <el-input v-model="inviteEmail" placeholder="email@example.com" style="width:260px" />
            <el-select v-model="inviteRole" style="width:120px">
              <el-option label="member" value="member" />
              <el-option label="admin" value="admin" />
            </el-select>
            <el-button type="primary" :loading="inviting" @click="doInvite">Invite</el-button>
          </div>
          <el-alert v-if="lastInviteUrl" type="success" show-icon style="margin-bottom:12px" :closable="true" @close="lastInviteUrl=''">
            <template #title>
              Invite link (copy now — token shown once):
              <code style="user-select:all">{{ lastInviteUrl }}</code>
              <el-button size="small" style="margin-left:8px" @click="copyInvite">Copy</el-button>
            </template>
          </el-alert>
          <h4 style="margin:8px 0">Members</h4>
          <el-table :data="members" class="dark-table" v-loading="loading" stripe style="margin-bottom:20px">
            <el-table-column prop="email" label="Email" />
            <el-table-column prop="name" label="Name" width="140" />
            <el-table-column prop="role" label="Role" width="100" />
            <el-table-column prop="joined_at" label="Joined" width="200" />
          </el-table>
          <h4 style="margin:8px 0">Pending invites</h4>
          <el-table :data="invites" class="dark-table" v-loading="loading" stripe>
            <el-table-column prop="email" label="Email" />
            <el-table-column prop="role" label="Role" width="100" />
            <el-table-column prop="expires_at" label="Expires" width="200" />
            <el-table-column label="Actions" width="120">
              <template #default="{row}">
                <el-button type="danger" size="small" @click="doRevoke(row.id)">Revoke</el-button>
              </template>
            </el-table-column>
          </el-table>
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

      <el-dialog v-model="renameVisible" :title="$t('team.rename')" width="480px">
        <el-form label-width="90px">
          <el-form-item :label="$t('dash.name')">
            <el-input v-model="renameName" />
          </el-form-item>
          <el-form-item :label="$t('dash.desc')">
            <el-input v-model="renameDesc" type="textarea" />
          </el-form-item>
          <p style="color:#909399;font-size:12px;margin:0 0 0 90px">{{ $t('dash.codeAutoHint') }}</p>
        </el-form>
        <template #footer>
          <el-button @click="renameVisible = false">{{ $t('dash.cancel') }}</el-button>
          <el-button type="primary" :loading="renaming" @click="doRename">{{ $t('dash.done') }}</el-button>
        </template>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { me, getWorkers, getLocks, getEvents, searchPlaybooks, getDAG, publishCommunityWorker, getLinkRequests, reviewLinkRequest, inviteMember, listInvites, revokeInvite, listMembers, listBranches, refreshBranches, listTeamDocs, listWorkerTemplates, patchBusinessProfile } from '@/api/hub'
import { useI18n } from '@/i18n'
import { parseTeamPath, buildTeamPath } from '@/utils/teamPath'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'
import RequirementsTab from '@/components/RequirementsTab.vue'
import KanbanBoard from '@/components/KanbanBoard.vue'

const route = useRoute()
const router = useRouter()
const { t: $t } = useI18n()

/** Stable short business_code for all API calls (not full URL slug). */
const teamCode = computed(() => {
  const seg = (route.params.slugCode as string) || (route.params.code as string) || ''
  return parseTeamPath(seg).code
})
const tab = ref('overview')
const team = ref<any>(null)
const workers = ref<any[]>([])
const locks = ref<any[]>([])
const playbooks = ref<any[]>([])
const events = ref<any[]>([])
const dag = ref<any[]>([])
const linkRequests = ref<any[]>([])
const members = ref<any[]>([])
const invites = ref<any[]>([])
const branches = ref<any[]>([])
const branchBindings = ref<any[]>([])
const refreshingBranches = ref(false)
const inviteEmail = ref('')
const inviteRole = ref('member')
const inviting = ref(false)
const lastInviteUrl = ref('')
const searchQ = ref('')
const pbDetail = ref<any>(null)
const pbVisible = ref(false)
const workerDetail = ref<any>(null)
const workerVisible = ref(false)
const loading = ref(false)
const teamDocs = ref<any[]>([])
const workerTemplates = ref<any[]>([])
function eventWho(e: any) { return e?.actor_email || e?.actor || '—' }
const errorMsg = ref('')
const sseConnected = ref(false)
const liveCount = ref(0)
let timer: number
let eventSource: EventSource | null = null

function isStale(row: any) { if (!row.last_heartbeat_at) return true; return Date.now() - new Date(row.last_heartbeat_at).getTime() > 90000 }
const staleCount = computed(() => workers.value.filter(isStale).length)

const branchRows = computed(() => {
  const binds = branchBindings.value || []
  return (branches.value || []).map((b: any) => {
    const related = binds.filter((x: any) => x.branch_name === b.name)
    return { ...b, bindings: related }
  })
})

async function doRefreshBranches() {
  const code = team.value?.code || teamCode.value
  if (!code) return
  refreshingBranches.value = true
  try {
    await refreshBranches(code)
    await loadTab()
    ElMessage.success('Branches refreshed')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || 'Refresh failed')
  }
  refreshingBranches.value = false
}

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
  const code = teamCode.value
  try {
    const bizRes = await me.getBusinesses()
    const list = bizRes.data?.data || []
    // match short code or still-valid alias (server may have reassigned)
    team.value = list.find((b: any) => b.code === code)
      || list.find((b: any) => parseTeamPath(String(route.params.slugCode || '')).code === b.code)
      || list.find((b: any) => code && (b.code === code || b.name === code))
    // if we only have legacy path without team, still try APIs with parsed code
    if (team.value && team.value.code) {
      const canonical = buildTeamPath(team.value.name, team.value.code)
      const cur = route.fullPath.split('?')[0]
      if (cur !== canonical && team.value.code.length === 4) {
        router.replace(canonical)
      }
    }
  } catch (e) { console.error(e) }
  loadTab()
}
async function loadTab() {
  const code = team.value?.code || teamCode.value
  if (!code) return
  loading.value = true; errorMsg.value = ''
  try {
    if (tab.value === 'overview' || tab.value === 'workers') { const r = await getWorkers({ business: code }); workers.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'locks') { const r = await getLocks({ business: code }); locks.value = r.data?.data || [] }
    if (tab.value === 'playbooks') { const r = await searchPlaybooks({ q: searchQ.value || '', business: code }); playbooks.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'events') { const r = await getEvents({ business: code, limit: 20 }); events.value = r.data?.data || [] }
    if (tab.value === 'overview' || tab.value === 'dag') { const r = await getDAG(code); dag.value = r.data?.data || [] }
    if (tab.value === 'docs') { try { const r = await listTeamDocs(code); teamDocs.value = r.data?.data || [] } catch { teamDocs.value = [] } }
    if (tab.value === 'overview' || tab.value === 'templates') { try { const r = await listWorkerTemplates(code); workerTemplates.value = r.data?.data || [] } catch { workerTemplates.value = [] } }
    if (tab.value === 'approvals') { const r = await getLinkRequests(code); linkRequests.value = r.data?.data || [] }
    if (tab.value === 'branches') {
      const r = await listBranches(code)
      branches.value = r.data?.data?.branches || []
      branchBindings.value = r.data?.data?.bindings || []
    }
    if (tab.value === 'members') {
      const [mRes, iRes] = await Promise.all([listMembers(code), listInvites(code)])
      members.value = mRes.data?.data || []
      invites.value = iRes.data?.data || []
    }
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
      business_code: teamCode.value,
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

async function reviewRequest(id: number, action: string) {
  try {
    await reviewLinkRequest(teamCode.value, id, action)
    ElMessage.success(action === 'approve' ? 'Request approved' : 'Request rejected')
    loadTab()
  } catch (e: any) { ElMessage.error(e?.message || 'Action failed') }
}

async function doInvite() {
  if (!inviteEmail.value) {
    ElMessage.warning('Email required')
    return
  }
  inviting.value = true
  try {
    const r = await inviteMember(teamCode.value, inviteEmail.value, inviteRole.value)
    lastInviteUrl.value = r.data?.data?.invite_url || ''
    ElMessage.success('Invite created — copy the link (token once)')
    inviteEmail.value = ''
    loadTab()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || 'Invite failed')
  }
  inviting.value = false
}

async function doRevoke(id: number) {
  try {
    await revokeInvite(teamCode.value, id)
    ElMessage.success('Invite revoked')
    loadTab()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || 'Revoke failed')
  }
}

function copyInvite() {
  if (!lastInviteUrl.value) return
  navigator.clipboard?.writeText(lastInviteUrl.value)
  ElMessage.success('Copied')
}

async function showWorker(row: any) {
  workerDetail.value = row; workerVisible.value = true; workerPlaybooks.value = []; workerEvents.value = []
  try {
    const [pRes, eRes] = await Promise.all([searchPlaybooks({ q: '', business: teamCode.value, limit: 50 }), getEvents({ business: teamCode.value, limit: 50 })])
    workerPlaybooks.value = (pRes.data?.data || []).filter((p: any) => p.created_by_worker_id === row.worker_id)
    workerEvents.value = (eRes.data?.data || []).filter((e: any) => e.actor === row.worker_id)
  } catch (e: any) { errorMsg.value = e?.message || 'Failed to load worker info' }
}

const renameVisible = ref(false)
const renameName = ref('')
const renameDesc = ref('')
const renaming = ref(false)

function openRename() {
  renameName.value = team.value?.name || ''
  renameDesc.value = team.value?.description || ''
  renameVisible.value = true
}

async function doRename() {
  if (!team.value?.code) return
  renaming.value = true
  try {
    const r = await patchBusinessProfile(team.value.code, {
      name: renameName.value.trim(),
      description: renameDesc.value,
    })
    const biz = r.data?.data?.business
    if (biz) {
      team.value = { ...team.value, ...biz }
      router.replace(buildTeamPath(biz.name, biz.code))
    }
    renameVisible.value = false
    ElMessage.success($t('team.renameOk'))
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || 'rename failed')
  }
  renaming.value = false
}

onMounted(() => {
  load()
  timer = window.setInterval(loadTab, 10000)
  connectSSE(teamCode.value)
})
onUnmounted(() => { clearInterval(timer); disconnectSSE() })
watch(tab, loadTab)
watch(() => route.params.slugCode, () => { load(); connectSSE(teamCode.value) })
</script>

<style>
.dark-table { --el-table-bg-color: #1a1a1a; --el-table-tr-bg-color: #1a1a1a; --el-table-header-bg-color: #1a1a1a; --el-table-border-color: #2a2a2a; --el-table-text-color: #ccc; --el-table-row-hover-bg-color: #2a2a2a; }
.dark-table .el-table__row--striped { --el-table-tr-bg-color: #222 !important; }
.pb-content { font-size:14px; line-height:1.8; color:#e0e0e0; }
.pb-content code { background:#2a2a2a; color:#67c23a; padding:2px 6px; border-radius:4px; font-size:13px; font-family:'Cascadia Code', 'Fira Code', monospace; }
.pb-dialog .el-dialog__body { padding-top:8px; }
</style>
