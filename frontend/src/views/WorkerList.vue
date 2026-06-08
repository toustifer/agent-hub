<template>
  <MainLayout>
    <div>
      <el-select v-model="filterBusiness" placeholder="Filter by business" clearable @change="load" style="width: 250px; margin-bottom: 16px">
        <el-option v-for="b in businesses" :key="b.code" :label="b.name" :value="b.code" />
      </el-select>
      <el-table :data="workers" stripe>
        <el-table-column prop="worker_id" label="Worker ID" width="180" />
        <el-table-column prop="version" label="Version" width="100" />
        <el-table-column prop="host" label="Host" width="150" />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'info'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_heartbeat_at" label="Last Heartbeat" width="200" />
        <el-table-column prop="pid" label="PID" width="100" />
      </el-table>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getWorkers, getBusinesses } from '@/api/hub'
import type { Worker, Business } from '@/types/hub'
import MainLayout from '@/layouts/MainLayout.vue'

const workers = ref<Worker[]>([])
const businesses = ref<Business[]>([])
const filterBusiness = ref('')

async function load() {
  const res = await getWorkers({ business: filterBusiness.value || undefined })
  workers.value = res.data?.data || []
}
onMounted(async () => {
  const bizRes = await getBusinesses()
  businesses.value = bizRes.data?.data || []
  load()
})
</script>
