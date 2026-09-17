<template>
  <div>
    <div style="display:flex;gap:12px;margin-bottom:16px;align-items:center;flex-wrap:wrap">
      <el-button type="primary" @click="showCreate = true">+ 提出需求</el-button>
      <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width:140px" @change="load">
        <el-option label="全部" value="" />
        <el-option label="草稿" value="draft" />
        <el-option label="已提交" value="submitted" />
        <el-option label="规划中" value="planning" />
        <el-option label="进行中" value="in_progress" />
        <el-option label="审核中" value="in_review" />
        <el-option label="已通过" value="accepted" />
        <el-option label="已打回" value="rejected" />
        <el-option label="已取消" value="cancelled" />
      </el-select>
    </div>

    <el-empty v-if="!loading && !list.length" description="暂无需求。点击「提出需求」创建第一个。" />
    <div v-else v-loading="loading">
      <el-card v-for="item in list" :key="item.id" shadow="hover" style="margin-bottom:10px;cursor:pointer" @click="expandId = expandId === item.id ? null : item.id">
        <div style="display:flex;align-items:center;gap:12px">
          <el-tag :type="statusTagType(item.status)">{{ item.status }}</el-tag>
          <span style="font-weight:bold;flex:1;font-size:15px">{{ item.title }}</span>
          <span style="color:#909399;font-size:12px">{{ item.created_by_email }}</span>
        </div>
        <div style="display:flex;align-items:center;gap:8px;margin-top:8px" v-if="item.task_count > 0">
          <el-progress :percentage="Math.round(item.task_count ? (item.tasks_done / item.task_count * 100) : 0)" :stroke-width="6" style="flex:1;max-width:300px" />
          <span style="color:#909399;font-size:12px">{{ item.tasks_done }}/{{ item.task_count }}</span>
        </div>
        <div style="margin-top:8px;display:flex;gap:8px" v-if="expandId !== item.id">
          <el-button size="small" v-if="item.status === 'draft'" @click.stop="editItem(item)">编辑</el-button>
          <el-button size="small" type="primary" v-if="item.status === 'draft'" @click.stop="doSubmit(item)">提交</el-button>
          <el-button size="small" type="danger" v-if="item.status === 'draft' || item.status === 'submitted'" @click.stop="doCancel(item)">取消</el-button>
          <el-button size="small" type="success" v-if="item.status === 'in_review'" @click.stop="doAccept(item)">通过</el-button>
          <el-button size="small" type="danger" v-if="item.status === 'in_review'" @click.stop="doReject(item)">打回</el-button>
        </div>

        <el-collapse-transition>
          <div v-if="expandId === item.id" style="margin-top:12px;padding-top:12px;border-top:1px solid #333">
            <div style="font-size:13px;color:#ccc;white-space:pre-wrap;margin-bottom:12px">{{ item.description || '（暂无描述）' }}</div>

            <!-- Linked tasks -->
            <div v-if="item.task_count > 0" style="margin-bottom:12px">
              <div style="font-weight:bold;font-size:13px;margin-bottom:6px">关联任务</div>
              <el-tag v-for="taskId in (detailTasks[item.id] || [])" :key="taskId" size="small" type="info" style="margin:2px">{{ taskId }}</el-tag>
            </div>

            <!-- Comments -->
            <div style="margin-bottom:8px">
              <div style="font-weight:bold;font-size:13px;margin-bottom:6px">评论</div>
              <div v-if="comments[item.id]?.length">
                <div v-for="c in comments[item.id]" :key="c.id" style="padding:6px 0;border-bottom:1px solid #2a2a2a;font-size:13px">
                  <span style="color:#409EFF">{{ c.author_email }}</span>
                  <el-tag v-if="c.decision" size="small" :type="tagType(c.decision)" style="margin:0 6px">{{ decisionLabel(c.decision) }}</el-tag>
                  <span style="color:#909399;font-size:11px">{{ c.created_at?.slice(0,16) }}</span>
                  <div style="margin-top:2px;color:#ccc">{{ c.body }}</div>
                </div>
              </div>
              <div v-else style="color:#909399;font-size:12px">暂无评论</div>
            </div>

            <div style="display:flex;gap:8px">
              <el-input v-model="commentBody" placeholder="写评论..." size="small" style="flex:1" />
              <el-button size="small" @click="addComment(item.id)">发送</el-button>
            </div>
          </div>
        </el-collapse-transition>
      </el-card>
    </div>

    <!-- Create dialog -->
    <el-dialog v-model="showCreate" title="提出需求" width="500px">
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="简要描述需求" :maxlength="512" />
        </el-form-item>
        <el-form-item label="详细描述">
          <el-input v-model="form.description" type="textarea" :rows="4" placeholder="详细说明（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">提交需求</el-button>
      </template>
    </el-dialog>

    <!-- Edit dialog -->
    <el-dialog v-model="showEdit" title="编辑需求" width="500px">
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="editForm.title" :maxlength="512" />
        </el-form-item>
        <el-form-item label="详细描述">
          <el-input v-model="editForm.description" type="textarea" :rows="4" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- Confirm dialog for submit/accept/reject/cancel -->
    <el-dialog v-model="showConfirm" :title="confirmTitle" width="450px">
      <el-input v-model="confirmComment" type="textarea" :rows="3" :placeholder="confirmPlaceholder" />
      <template #footer>
        <el-button @click="showConfirm = false">取消</el-button>
        <el-button :type="confirmType" :loading="saving" @click="doConfirm">{{ confirmAction }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  listRequirements, getRequirement, createRequirement, updateRequirement,
  submitRequirement, acceptRequirement, rejectRequirement, cancelRequirement,
  listRequirementComments, createRequirementComment,
} from '@/api/hub'
import { ElMessage } from 'element-plus'

const props = defineProps<{ businessCode: string }>()

const list = ref<any[]>([])
const loading = ref(false)
const statusFilter = ref('')
const expandId = ref<number | null>(null)
const comments = ref<Record<number, any[]>>({})
const detailTasks = ref<Record<number, string[]>>({})
const commentBody = ref('')

// Create
const showCreate = ref(false)
const form = ref({ title: '', description: '' })
const saving = ref(false)

// Edit
const showEdit = ref(false)
const editForm = ref({ title: '', description: '' })
const editTarget = ref<any>(null)

// Confirm actions
const showConfirm = ref(false)
const confirmAction = ref('')
const confirmTitle = ref('')
const confirmType = ref<'primary' | 'success' | 'danger' | 'warning'>('primary')
const confirmPlaceholder = ref('')
const confirmCallback = ref<() => Promise<void>>(async () => {})
const confirmComment = ref('')

const statusMap: Record<string, { label: string; type: string }> = {
  draft: { label: '草稿', type: 'info' },
  submitted: { label: '已提交', type: 'primary' },
  planning: { label: '规划中', type: 'warning' },
  in_progress: { label: '进行中', type: 'warning' },
  in_review: { label: '审核中', type: 'primary' },
  accepted: { label: '已通过', type: 'success' },
  rejected: { label: '已打回', type: 'danger' },
  cancelled: { label: '已取消', type: 'info' },
}

function statusTagType(status: string) {
  switch (status) {
    case 'draft': return 'info'
    case 'submitted': return 'warning'
    case 'in_review': return 'primary'
    case 'accepted': return 'success'
    case 'rejected': return 'danger'
    case 'cancelled': return 'info'
    default: return 'info'
  }
}

function tagType(s: string) { return statusMap[s]?.type || 'info' }
function decisionLabel(d: string) {
  return { accept: '通过', reject: '打回', changes_requested: '请求修改' }[d] || d
}

async function load() {
  loading.value = true
  try {
    const r = await listRequirements(props.businessCode, statusFilter.value || undefined)
    list.value = r.data?.data || []
  } catch (e: any) { ElMessage.error(e?.message || '加载失败') }
  loading.value = false
}

async function loadComments(id: number) {
  if (comments.value[id]) return
  try {
    const r = await listRequirementComments(props.businessCode, id)
    comments.value[id] = r.data?.data || []
  } catch {}
}

async function loadDetail(id: number) {
  try {
    const r = await getRequirement(props.businessCode, id)
    detailTasks.value[id] = r.data?.data?.task_ids || []
  } catch {}
}

async function doCreate() {
  if (!form.value.title.trim()) { ElMessage.warning('请填写标题'); return }
  saving.value = true
  try {
    await createRequirement(props.businessCode, {
      title: form.value.title.trim(),
      description: form.value.description,
    })
    ElMessage.success('需求已创建')
    showCreate.value = false
    form.value = { title: '', description: '' }
    load()
  } catch (e: any) { ElMessage.error(e?.response?.data?.message || e?.message || '创建失败') }
  saving.value = false
}

function editItem(item: any) {
  editTarget.value = item
  editForm.value = { title: item.title, description: item.description }
  showEdit.value = true
}

async function doEdit() {
  if (!editForm.value.title.trim()) { ElMessage.warning('请填写标题'); return }
  saving.value = true
  try {
    await updateRequirement(props.businessCode, editTarget.value.id, {
      title: editForm.value.title.trim(),
      description: editForm.value.description,
    })
    ElMessage.success('已保存')
    showEdit.value = false
    load()
  } catch (e: any) { ElMessage.error(e?.response?.data?.message || e?.message || '保存失败') }
  saving.value = false
}

function confirmDialog(title: string, action: string, type: 'primary' | 'success' | 'danger', placeholder: string, cb: () => Promise<void>) {
  confirmTitle.value = title
  confirmAction.value = action
  confirmType.value = type
  confirmPlaceholder.value = placeholder
  confirmComment.value = ''
  confirmCallback.value = cb
  showConfirm.value = true
}

function doSubmit(item: any) {
  confirmDialog('提交需求', '提交', 'primary', '提交备注（可选）', async () => {
    await submitRequirement(props.businessCode, item.id, confirmComment.value || undefined)
    ElMessage.success('已提交')
    showConfirm.value = false
    load()
  })
}

function doAccept(item: any) {
  confirmDialog('通过需求', '通过', 'success', '验收意见（可选）', async () => {
    await acceptRequirement(props.businessCode, item.id, confirmComment.value || undefined)
    ElMessage.success('已通过')
    showConfirm.value = false
    load()
  })
}

function doReject(item: any) {
  confirmDialog('打回需求', '打回', 'danger', '打回原因（可选）', async () => {
    await rejectRequirement(props.businessCode, item.id, confirmComment.value || undefined)
    ElMessage.success('已打回')
    showConfirm.value = false
    load()
  })
}

function doCancel(item: any) {
  confirmDialog('取消需求', '取消', 'danger', '取消原因（可选）', async () => {
    await cancelRequirement(props.businessCode, item.id, confirmComment.value || undefined)
    ElMessage.success('已取消')
    showConfirm.value = false
    load()
  })
}

async function doConfirm() {
  saving.value = true
  try { await confirmCallback.value() } catch (e: any) { ElMessage.error(e?.response?.data?.message || e?.message || '操作失败') }
  saving.value = false
}

async function addComment(id: number) {
  if (!commentBody.value.trim()) return
  try {
    await createRequirementComment(props.businessCode, id, { body: commentBody.value })
    commentBody.value = ''
    delete comments.value[id]
    await loadComments(id)
  } catch (e: any) { ElMessage.error(e?.response?.data?.message || e?.message || '评论失败') }
}

// On expand, load comments
watch(expandId, (id) => { if (id) { loadComments(id); loadDetail(id) } })

// Load on mount and when businessCode changes
watch(() => props.businessCode, load, { immediate: true })
</script>

<style scoped>
.el-card { background: #1a1a1a; border-color: #2a2a2a; }
.el-card:hover { border-color: #409EFF; }
</style>
