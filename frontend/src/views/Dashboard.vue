<template>
  <MainLayout>
    <div>
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
        <h2>{{ $t('dash.title') }}</h2>
        <el-button type="primary" @click="showCreate = true">{{ $t('dash.newTeam') }}</el-button>
      </div>

      <el-alert
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom:16px"
        :title="$t('dash.codeMigratedTitle')"
      >
        <div>{{ $t('dash.codeMigratedHint') }}</div>
      </el-alert>

      <el-row :gutter="16" style="margin-bottom:24px" v-if="teams.length">
        <el-col :span="6" v-for="s in statsCards" :key="s.label">
          <el-card shadow="never" style="text-align:center;background:linear-gradient(135deg,#1a1a2e,#16213e);border:none">
            <div style="font-size:28px;font-weight:bold;color:#409EFF">{{ s.value }}</div>
            <div style="color:#909399;font-size:13px;margin-top:4px">{{ s.label }}</div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="20" v-if="teams.length">
        <el-col :span="8" v-for="item in teams" :key="item.id" style="margin-bottom:20px">
          <el-card shadow="hover" style="cursor:pointer">
            <div style="display:flex;align-items:center;gap:12px;margin-bottom:12px" @click="goTeam(item)">
              <div style="width:48px;height:48px;border-radius:8px;background:#409EFF;color:#fff;display:flex;align-items:center;justify-content:center;font-size:20px;font-weight:bold">{{ (item.name || item.code || '?')[0] }}</div>
              <div style="flex:1">
                <div style="font-weight:bold;font-size:16px">{{ item.name }}</div>
                <div style="color:#909399;font-size:13px">{{ $t('dash.codeLabel') }}: <code>{{ item.code }}</code></div>
              </div>
            </div>
            <p style="color:#606266;font-size:14px;margin-bottom:12px;min-height:40px">{{ item.description || '—' }}</p>
            <div style="display:flex;gap:6px;align-items:center">
              <el-tag :type="item.status === 'active' ? 'success' : 'warning'" size="small">{{ item.status }}</el-tag>
              <el-tag type="info" size="small">{{ item.role }}</el-tag>
              <span style="flex:1" />
              <el-button size="small" type="primary" text @click="goTeam(item)">{{ $t('dash.viewDetail') }}</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <el-empty v-else :description="$t('dash.noTeams')" />

      <el-dialog v-model="showCreate" :title="$t('dash.newTeam')" width="500px">
        <el-form :model="form" label-width="100px">
          <el-form-item :label="$t('dash.name')"><el-input v-model="form.name" :placeholder="$t('dash.namePlaceholder')" /></el-form-item>
          <el-form-item :label="$t('dash.desc')"><el-input v-model="form.desc" type="textarea" /></el-form-item>
          <p style="color:#909399;font-size:12px;margin:0 0 8px 100px">{{ $t('dash.codeAutoHint') }}</p>
        </el-form>
        <template #footer>
          <el-button @click="showCreate = false">{{ $t('dash.cancel') }}</el-button>
          <el-button type="primary" @click="createTeam" :loading="creating">{{ $t('dash.create') }}</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="showCreated" :title="$t('dash.created')" width="520px">
        <el-alert type="success" :title="$t('dash.createdHint')" :closable="false" style="margin-bottom:16px" />
        <p style="margin:0 0 8px;font-size:14px">
          <strong>{{ $t('dash.codeLabel') }}:</strong>
          <code style="margin-left:8px;user-select:all;font-size:16px">{{ newTeamCode }}</code>
          <el-button size="small" style="margin-left:8px" @click="copyCode">{{ $t('docs.copy') }}</el-button>
        </p>
        <p style="color:#606266;font-size:13px;line-height:1.6">{{ $t('dash.connectInfo') }}</p>
        <p style="color:#909399;font-size:12px">{{ $t('dash.softSyncHint') }}</p>
        <template #footer>
          <el-button @click="showCreated=false">{{ $t('dash.done') }}</el-button>
          <el-button type="primary" @click="openNewTeam">{{ $t('dash.viewDetail') }}</el-button>
        </template>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { me, createBusiness } from '@/api/hub'
import { useI18n } from '@/i18n'
import { buildTeamPath } from '@/utils/teamPath'
import { ElMessage } from 'element-plus'
import MainLayout from '@/layouts/MainLayout.vue'

const router = useRouter()
const { t: $t, locale } = useI18n()
const teams = ref<any[]>([])

const statsCards = computed(() => {
  const active = teams.value.filter(t => t.status === 'active').length
  return [
    { label: locale.value === 'zh' ? '团队总数' : 'Total Teams', value: teams.value.length },
    { label: locale.value === 'zh' ? '活跃团队' : 'Active', value: active },
    { label: locale.value === 'zh' ? '角色' : 'Roles', value: [...new Set(teams.value.map(t => t.role))].length },
    { label: locale.value === 'zh' ? '成员' : 'Members', value: '—' },
  ]
})
const showCreate = ref(false)
const showCreated = ref(false)
const newTeamCode = ref('')
const newTeamName = ref('')
const creating = ref(false)
const form = reactive({ name: '', desc: '' })

onMounted(async () => {
  try { const res = await me.getBusinesses(); teams.value = res.data?.data || [] } catch (e) { console.error(e) }
})

function goTeam(item: { name?: string; code: string }) {
  router.push(buildTeamPath(item.name || '', item.code))
}

async function createTeam() {
  if (!form.name.trim()) {
    ElMessage.warning($t('dash.nameRequired'))
    return
  }
  creating.value = true
  try {
    const res = await createBusiness({ name: form.name.trim(), description: form.desc })
    showCreate.value = false
    const biz = res.data?.data?.business
    newTeamCode.value = biz?.code || ''
    newTeamName.value = biz?.name || form.name
    form.name = ''; form.desc = ''
    showCreated.value = true
    const r = await me.getBusinesses(); teams.value = r.data?.data || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || 'create failed')
  }
  creating.value = false
}

function openNewTeam() {
  showCreated.value = false
  if (newTeamCode.value) router.push(buildTeamPath(newTeamName.value, newTeamCode.value))
}

async function copyCode() {
  try {
    await navigator.clipboard.writeText(newTeamCode.value)
    ElMessage.success($t('docs.copied'))
  } catch { /* ignore */ }
}
</script>
