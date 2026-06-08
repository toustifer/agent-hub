<template>
  <MainLayout>
    <div>
      <!-- Header -->
      <h2 style="margin:0 0 20px">{{ $t('community.title') }}</h2>

      <el-alert v-if="errorMsg" :title="errorMsg" type="error" show-icon style="margin-bottom:12px" />

      <!-- Search & Filters -->
      <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap">
        <el-input
          v-model="searchQ"
          :placeholder="$t('community.search')"
          style="width:320px"
          clearable
          :prefix-icon="Search"
          @input="onSearchInput"
        />
        <el-select v-model="domainFilter" :placeholder="$t('community.domainFilter')" style="width:160px" @change="load">
          <el-option :label="$t('community.allDomains')" value="" />
          <el-option label="frontend" value="frontend" />
          <el-option label="backend" value="backend" />
          <el-option label="testing" value="testing" />
          <el-option label="ble" value="ble" />
          <el-option label="devops" value="devops" />
          <el-option label="ai" value="ai" />
          <el-option label="hardware" value="hardware" />
          <el-option label="other" value="other" />
        </el-select>
        <el-select v-model="sortBy" style="width:160px" @change="load">
          <el-option :label="$t('community.popular')" value="popular" />
          <el-option :label="$t('community.latest')" value="latest" />
        </el-select>
      </div>

      <!-- Loading / Empty / Error states -->
      <div v-loading="loading">
        <el-empty v-if="!loading && !workers.length && !errorMsg" :description="$t('community.noWorkers')" />

        <!-- Worker Cards Grid -->
        <el-row v-if="workers.length" :gutter="16">
          <el-col v-for="w in workers" :key="w.id" :span="8" style="margin-bottom:16px">
            <el-card shadow="hover" class="worker-card" @click="goDetail(w.id)">
              <!-- Domain Tag -->
              <div style="margin-bottom:8px">
                <el-tag :color="domainColor(w.domain)" size="small" effect="dark" style="border:none">
                  {{ w.domain || 'other' }}
                </el-tag>
              </div>
              <!-- Title -->
              <div class="worker-title" style="font-weight:bold;font-size:15px;margin-bottom:8px;cursor:pointer">
                {{ w.title || w.worker_id }}
              </div>
              <!-- Description -->
              <div class="worker-desc" style="font-size:13px;color:#909399;margin-bottom:12px;line-height:1.5;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden">
                {{ w.description || $t('team.noDescription') }}
              </div>
              <!-- Tags -->
              <div v-if="w.tags && w.tags.length" style="margin-bottom:8px;display:flex;gap:4px;flex-wrap:wrap">
                <el-tag v-for="tag in w.tags" :key="tag" size="small" type="info">{{ tag }}</el-tag>
              </div>
              <!-- Bottom row: installs + status -->
              <div style="display:flex;align-items:center;justify-content:space-between">
                <span style="font-size:12px;color:#909399">
                  <el-icon style="margin-right:2px;vertical-align:middle"><Download /></el-icon>
                  {{ w.install_count || 0 }} {{ $t('community.installs') }}
                </span>
                <el-tag size="small" :type="w.status === 'published' ? 'success' : 'warning'">
                  {{ w.status || 'published' }}
                </el-tag>
              </div>
            </el-card>
          </el-col>
        </el-row>

        <!-- Pagination -->
        <div v-if="total > pageSize" style="margin-top:20px;text-align:center">
          <el-pagination
            v-model:current-page="page"
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            @current-change="load"
          />
        </div>
      </div>
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCommunityWorkers } from '@/api/hub'
import { useI18n } from '@/i18n'
import { Search, Download } from '@element-plus/icons-vue'
import MainLayout from '@/layouts/MainLayout.vue'

const router = useRouter()
const { t: $t } = useI18n()

const workers = ref<any[]>([])
const searchQ = ref('')
const domainFilter = ref('')
const sortBy = ref('popular')
const loading = ref(false)
const errorMsg = ref('')
const page = ref(1)
const pageSize = 24
const total = ref(0)

let searchTimer: number

function domainColor(domain: string): string {
  const colors: Record<string, string> = {
    frontend: '#409EFF', backend: '#67C23A', testing: '#E6A23C',
    ble: '#F56C6C', devops: '#909399', ai: '#9B59B6',
    hardware: '#1ABC9C', other: '#95A5A6'
  }
  return colors[domain] || '#95A5A6'
}

function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => { page.value = 1; load() }, 300)
}

function goDetail(id: number) {
  router.push(`/community/${id}`)
}

async function load() {
  loading.value = true
  errorMsg.value = ''
  try {
    const params: any = { page: page.value, page_size: pageSize, sort: sortBy.value }
    if (domainFilter.value) params.domain = domainFilter.value
    if (searchQ.value) params.search = searchQ.value
    const res = await listCommunityWorkers(params)
    workers.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (e: any) {
    errorMsg.value = e?.message || 'Failed to load'
    workers.value = []
  }
  loading.value = false
}

onMounted(() => load())
</script>

<style>
.worker-card {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}
.worker-card:hover {
  transform: translateY(-2px);
}

/* Dark mode overrides for community cards */
html.dark .worker-card {
  background: #1e1e1e !important;
  border-color: #333 !important;
}
html.dark .worker-title {
  color: #e0e0e0;
}
html.dark .worker-desc {
  color: #909399;
}
html.dark .el-empty__description {
  color: #909399;
}
html.dark .el-pagination .el-pager li {
  background: #252525 !important;
  color: #ccc !important;
}
html.dark .el-pagination .el-pager li.is-active {
  background: #409EFF !important;
  color: #fff !important;
}
html.dark .el-select__wrapper {
  background: #252525 !important;
}
html.dark .el-select-dropdown__item {
  color: #ccc !important;
}
html.dark .el-select-dropdown__item:hover {
  background: #2a2a2a !important;
}
html.dark .el-popper.is-light {
  background: #1e1e1e !important;
  border-color: #333 !important;
}
</style>
