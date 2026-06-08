<template>
  <MainLayout>
    <div>
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
        <h2>{{ $t('dash.title') }}</h2>
        <el-button type="primary" @click="showCreate = true">{{ $t('dash.newTeam') }}</el-button>
      </div>

      <!-- Stats overview -->
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
            <div style="display:flex;align-items:center;gap:12px;margin-bottom:12px" @click="$router.push('/team/' + item.code)">
              <div style="width:48px;height:48px;border-radius:8px;background:#409EFF;color:#fff;display:flex;align-items:center;justify-content:center;font-size:20px;font-weight:bold">{{ item.name[0] }}</div>
              <div style="flex:1">
                <div style="font-weight:bold;font-size:16px">{{ item.name }}</div>
                <div style="color:#909399;font-size:13px">{{ item.code }}</div>
              </div>
            </div>
            <p style="color:#606266;font-size:14px;margin-bottom:12px;min-height:40px">{{ item.description || '—' }}</p>
            <div style="display:flex;gap:6px;align-items:center">
              <el-tag :type="item.status === 'active' ? 'success' : 'warning'" size="small">{{ item.status }}</el-tag>
              <el-tag type="info" size="small">{{ item.role }}</el-tag>
              <span style="flex:1" />
              <el-button size="small" type="primary" text @click="$router.push('/team/' + item.code)">{{ $t('dash.viewDetail') }}</el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <el-empty v-else :description="$t('dash.noTeams')" />

      <el-dialog v-model="showCreate" :title="$t('dash.newTeam')" width="500px">
        <el-form :model="form" label-width="100px">
          <el-form-item :label="$t('dash.code')"><el-input v-model="form.code" :placeholder="$t('dash.codePlaceholder')" /></el-form-item>
          <el-form-item :label="$t('dash.name')"><el-input v-model="form.name" :placeholder="$t('dash.namePlaceholder')" /></el-form-item>
          <el-form-item :label="$t('dash.desc')"><el-input v-model="form.desc" type="textarea" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showCreate = false">{{ $t('dash.cancel') }}</el-button>
          <el-button type="primary" @click="createTeam" :loading="creating">{{ $t('dash.create') }}</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="showApiKey" :title="$t('dash.created')" width="500px">
        <el-alert type="success" :title="$t('dash.saveKey')" :closable="false" style="margin-bottom:16px" />
        <el-input v-model="apiKey" readonly size="large" />
        <p style="color:#909399;font-size:13px;margin-top:12px">{{ $t('dash.connectInfo') }}</p>
        <template #footer><el-button type="primary" @click="showApiKey=false;apiKey=''">{{ $t('dash.done') }}</el-button></template>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { me, createBusiness } from '@/api/hub'
import { useI18n } from '@/i18n'
import MainLayout from '@/layouts/MainLayout.vue'

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
const showApiKey = ref(false)
const apiKey = ref('')
const creating = ref(false)
const form = reactive({ code: '', name: '', desc: '' })

onMounted(async () => {
  try { const res = await me.getBusinesses(); teams.value = res.data?.data || [] } catch (e) { console.error(e) }
})

async function createTeam() {
  creating.value = true
  try {
    const res = await createBusiness({ code: form.code, name: form.name, description: form.desc })
    showCreate.value = false
    form.code = ''; form.name = ''; form.desc = ''
    if (res.data?.data?.api_key) { apiKey.value = res.data.data.api_key; showApiKey.value = true }
    const r = await me.getBusinesses(); teams.value = r.data?.data || []
  } catch (e) { console.error(e) }
  creating.value = false
}
</script>
