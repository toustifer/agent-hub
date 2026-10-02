<template>
  <div>
    <div style="display:flex;gap:12px;overflow-x:auto;padding-bottom:12px;min-height:400px">
      <div v-for="col in columns" :key="col.status" style="flex:1;min-width:220px;background:#141414;border-radius:8px;padding:12px">
        <div style="font-weight:bold;font-size:14px;margin-bottom:12px;color:#e0e0e0">
          {{ col.label }}
          <span style="color:#909399;font-size:12px;margin-left:6px">({{ col.tasks.length }})</span>
        </div>
        <div v-if="col.tasks.length === 0" style="color:#555;font-size:13px;text-align:center;padding:20px 0">空</div>
        <el-card v-for="task in col.tasks" :key="task.task_id" shadow="never" style="margin-bottom:8px;background:#1a1a1a;border-color:#2a2a2a;cursor:default">
          <div style="font-size:13px;font-weight:bold;color:#e0e0e0">{{ task.title }}</div>
          <div style="margin-top:6px;display:flex;align-items:center;gap:6px" v-if="task.status">
            <el-tag :type="taskStatusTagType(task.status)" size="small" effect="plain">{{ task.status }}</el-tag>
          </div>
          <div style="font-size:11px;color:#909399;margin-top:4px">{{ task.task_id }}</div>
          <div style="font-size:11px;color:#67c23a;margin-top:2px" v-if="task.assignee_email || task.assigned_worker">
            {{ task.assignee_email || task.assigned_worker }}
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getDAG } from '@/api/hub'
import { ElMessage } from 'element-plus'

const props = defineProps<{ businessCode: string }>()

const tasks = ref<any[]>([])

function mapStatusToColumn(status: string): string {
  const s = (status || '').toLowerCase().trim()
  switch (s) {
    case 'completed':
    case 'done':
    case 'passed':
      return 'completed'
    case 'in_progress':
    case 'executing':
    case 'rework_needed':
      return 'in_progress'
    case 'in_review':
    case 'review_pending':
      return 'in_review'
    case 'pending':
    case 'assigned':
    default:
      return 'pending'
  }
}

function taskStatusTagType(status: string) {
  const s = (status || '').toLowerCase().trim()
  switch (s) {
    case 'completed':
    case 'passed':
    case 'done':
      return 'success'
    case 'in_progress':
    case 'executing':
      return 'warning'
    case 'failed':
    case 'blocked':
    case 'rework_needed':
      return 'danger'
    case 'review_pending':
    case 'in_review':
      return 'primary'
    case 'assigned':
    case 'pending':
    default:
      return 'info'
  }
}

const columns = computed(() => {
  const colDefs = [
    { status: 'pending', label: '待办', tasks: [] as any[] },
    { status: 'in_progress', label: '进行中', tasks: [] as any[] },
    { status: 'in_review', label: '审核中', tasks: [] as any[] },
    { status: 'completed', label: '已完成', tasks: [] as any[] },
  ]
  for (const t of tasks.value) {
    const colKey = mapStatusToColumn(t.status)
    const col = colDefs.find(c => c.status === colKey) || colDefs[0]
    col.tasks.push(t)
  }
  return colDefs
})

async function load() {
  if (!props.businessCode) return
  try {
    const r = await getDAG(props.businessCode)
    tasks.value = r.data?.data || []
  } catch (e: any) { ElMessage.error(e?.message || '加载失败') }
}

watch(() => props.businessCode, load, { immediate: true })
</script>
