<template>
  <MainLayout>
    <div>
      <el-select v-model="filterBusiness" placeholder="Filter by business" clearable @change="load" style="width: 250px; margin-bottom: 16px">
        <el-option v-for="b in businesses" :key="b.code" :label="b.name" :value="b.code" />
      </el-select>
      <el-table :data="locks" stripe>
        <el-table-column prop="resource_key" label="Resource Key" width="300" />
        <el-table-column prop="holder_worker_id" label="Holder Worker" width="200" />
        <el-table-column prop="acquired_at" label="Acquired At" width="200" />
        <el-table-column prop="expires_at" label="Expires At" width="200" />
      </el-table>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getLocks, getBusinesses } from '@/api/hub'
import type { Lock, Business } from '@/types/hub'
import MainLayout from '@/layouts/MainLayout.vue'

const locks = ref<Lock[]>([])
const businesses = ref<Business[]>([])
const filterBusiness = ref('')
let timer: number

async function load() {
  const res = await getLocks({ business: filterBusiness.value || undefined })
  locks.value = res.data?.data || []
}
onMounted(async () => {
  const bizRes = await getBusinesses()
  businesses.value = bizRes.data?.data || []
  load()
  timer = window.setInterval(load, 10000)
})
onUnmounted(() => clearInterval(timer))
</script>
