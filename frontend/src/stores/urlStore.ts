import { defineStore } from 'pinia'
import { ref } from 'vue'
import { urlService } from '../services/urlService'
import type { ShortURL } from '../types'

export const useUrlStore = defineStore('url', () => {
  const urls = ref<ShortURL[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref('')

  async function fetchAll() {
    loading.value = true
    error.value = ''
    try {
      const resp = await urlService.list()
      if (resp.success && resp.data) {
        urls.value = resp.data.urls
        total.value = resp.data.total
      }
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || 'Failed to load URLs'
    } finally {
      loading.value = false
    }
  }

  async function create(originalUrl: string): Promise<ShortURL | null> {
    error.value = ''
    try {
      const resp = await urlService.create(originalUrl)
      if (resp.success && resp.data) {
        urls.value = [resp.data, ...urls.value]
        total.value += 1
        return resp.data
      }
      return null
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || 'Failed to create URL'
      throw err
    }
  }

  async function remove(id: string) {
    error.value = ''
    try {
      const resp = await urlService.remove(id)
      if (resp.success) {
        urls.value = urls.value.filter((u) => u.id !== id)
        total.value = Math.max(0, total.value - 1)
      }
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || 'Failed to delete URL'
      throw err
    }
  }

  return { urls, total, loading, error, fetchAll, create, remove }
})
