<template>
  <MainLayout>
    <div>
      <el-row :gutter="16" style="margin-bottom: 16px">
        <el-col :span="12">
          <el-input v-model="query" placeholder="Search playbooks..." @keyup.enter="search" clearable />
        </el-col>
        <el-col :span="6">
          <el-select v-model="filterCategory" placeholder="Category" clearable @change="search">
            <el-option label="Decisions" value="decisions" />
            <el-option label="Patterns" value="patterns" />
            <el-option label="Gotchas" value="gotchas" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="search">Search</el-button>
        </el-col>
      </el-row>
      <el-table :data="playbooks" stripe @row-click="showDetail">
        <el-table-column prop="title" label="Title" width="250" />
        <el-table-column prop="category" label="Category" width="120">
          <template #default="{ row }">
            <el-tag>{{ row.category }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tags" label="Tags" width="200">
          <template #default="{ row }">
            <el-tag v-for="tag in row.tags" :key="tag" size="small" style="margin-right: 4px">{{ tag }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_by_worker_id" label="Author" width="150" />
      </el-table>
      <el-dialog v-model="dialogVisible" :title="selected?.title" width="700px">
        <pre style="white-space: pre-wrap; word-break: break-word">{{ selected?.content }}</pre>
      </el-dialog>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { searchPlaybooks } from '@/api/hub'
import type { Playbook } from '@/types/hub'
import MainLayout from '@/layouts/MainLayout.vue'

const query = ref('')
const filterCategory = ref('')
const playbooks = ref<Playbook[]>([])
const dialogVisible = ref(false)
const selected = ref<Playbook | null>(null)

async function search() {
  const res = await searchPlaybooks({ q: query.value || undefined, category: filterCategory.value || undefined })
  playbooks.value = res.data?.data || []
}
function showDetail(row: Playbook) {
  selected.value = row
  dialogVisible.value = true
}
</script>
