<template>
  <MainLayout>
    <div>
      <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px">
        <el-select v-model="filterBusiness" placeholder="Business" clearable @change="reconnect" style="width: 200px">
          <el-option v-for="b in businesses" :key="b.code" :label="b.name" :value="b.code" />
        </el-select>
        <el-tag :type="connected ? 'success' : 'danger'">{{ connected ? 'Connected' : 'Disconnected' }}</el-tag>
        <el-button v-if="connected" @click="pause">Pause</el-button>
        <el-button v-else @click="resume">Resume</el-button>
      </div>
      <el-timeline>
        <el-timeline-item
          v-for="event in events"
          :key="event.id"
          :timestamp="event.created_at"
          placement="top"
        >
          <el-card>
            <strong>{{ event.event_type }}</strong> by {{ event.actor }}
            <p style="margin-top: 8px; color: #909399">{{ JSON.stringify(event.payload) }}</p>
          </el-card>
        </el-timeline-item>
      </el-timeline>
      <el-empty v-if="events.length === 0 && connected" description="Waiting for events..." />
    </div>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getBusinesses } from '@/api/hub'
import type { Business, HubEvent } from '@/types/hub'
import MainLayout from '@/layouts/MainLayout.vue'

const events = ref<HubEvent[]>([])
const businesses = ref<Business[]>([])
const filterBusiness = ref('')
const connected = ref(false)
let eventSource: EventSource | null = null
let paused = false
let queue: HubEvent[] = []

function connect() {
  const base = import.meta.env.VITE_HUB_API || 'https://hub.stifer.xyz'
  const token = localStorage.getItem('token')
  const params = new URLSearchParams()
  if (filterBusiness.value) params.set('business', filterBusiness.value)
  const url = `${base}/v1/hub/events/stream?${params.toString()}`

  eventSource = new EventSource(url + '&token=' + token)

  eventSource.onopen = () => { connected.value = true }
  eventSource.onerror = () => {
    connected.value = false
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  }
  eventSource.addEventListener('message', (e) => {
    const data = JSON.parse(e.data)
    if (paused) {
      queue.push(data)
    } else {
      events.value.unshift(data)
      if (events.value.length > 100) events.value.pop()
    }
  })
}

function disconnect() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
  connected.value = false
}

function reconnect() { disconnect(); connect() }
function pause() { paused = true }
function resume() {
  paused = false
  while (queue.length) events.value.unshift(queue.shift()!)
  if (events.value.length > 100) events.value = events.value.slice(0, 100)
}

onMounted(async () => {
  const bizRes = await getBusinesses()
  businesses.value = bizRes.data?.data || []
  connect()
})

onUnmounted(() => disconnect())
</script>
