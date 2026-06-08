import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getBusinesses } from '@/api/hub'
import type { Business } from '@/types/hub'

export const useAppStore = defineStore('app', () => {
  const businesses = ref<Business[]>([])
  const currentBusiness = ref('')

  async function fetchBusinesses() {
    try {
      const res = await getBusinesses()
      businesses.value = res.data?.data || []
    } catch (e) {
      console.error('fetch businesses failed', e)
    }
  }

  return { businesses, currentBusiness, fetchBusinesses }
})
