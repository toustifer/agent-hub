<template>
  <MainLayout>
    <div>
      <div style="display: flex; justify-content: space-between; margin-bottom: 16px">
        <el-select v-model="filterStatus" placeholder="Filter by status" clearable @change="load" style="width: 200px">
          <el-option label="Active" value="active" />
          <el-option label="Pending" value="pending" />
          <el-option label="Suspended" value="suspended" />
        </el-select>
        <el-button type="primary" @click="dialogVisible = true">New Business</el-button>
      </div>
      <el-table :data="businesses" stripe>
        <el-table-column prop="code" label="Code" width="150" />
        <el-table-column prop="name" label="Name" width="200" />
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : row.status === 'pending' ? 'warning' : 'danger'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="owner_user_id" label="Owner ID" width="120" />
        <el-table-column prop="created_at" label="Created" width="200" />
        <el-table-column prop="description" label="Description" />
      </el-table>
      <el-dialog v-model="dialogVisible" title="New Business" width="500px">
        <el-form :model="form" label-width="120px">
          <el-form-item label="Code"><el-input v-model="form.code" /></el-form-item>
          <el-form-item label="Name"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="Repo URL"><el-input v-model="form.repo_url" /></el-form-item>
          <el-form-item label="Owner User ID"><el-input-number v-model="form.owner_user_id" /></el-form-item>
          <el-form-item label="Description"><el-input v-model="form.description" type="textarea" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogVisible = false">Cancel</el-button>
          <el-button type="primary" @click="create">Create</el-button>
        </template>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { getBusinesses, createBusiness } from '@/api/hub'
import type { Business } from '@/types/hub'
import MainLayout from '@/layouts/MainLayout.vue'

const businesses = ref<Business[]>([])
const filterStatus = ref('')
const dialogVisible = ref(false)
const form = reactive({ code: '', name: '', repo_url: '', owner_user_id: 0, description: '' })

async function load() {
  const res = await getBusinesses({ status: filterStatus.value || undefined })
  businesses.value = res.data?.data || []
}
async function create() {
  await createBusiness(form)
  dialogVisible.value = false
  Object.assign(form, { code: '', name: '', repo_url: '', owner_user_id: 0, description: '' })
  load()
}
onMounted(load)
</script>
