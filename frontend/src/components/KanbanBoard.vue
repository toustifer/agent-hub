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

const columns = computed(() => {
  const colDefs = [
    { status: 'pending', label: '待办', tasks: [] as any[] },
    { status: 'in_progress', label: '进行中', tasks: [] as any[] },
    { status: 'in_review', label: '审核中', tasks: [] as any[] },
    { status: 'completed', label: '已完成', tasks: [] as any[] },
  ]
  for (const t of tasks.value) {
    const col = colDefs.find(c => c.status === t.status) || colDefs[0]
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
