import { defineStore } from 'pinia'
import { ref } from 'vue'
import { urlService } from '../services/urlService'
import { sseService } from '../services/sseService'
import type { ShortURL, SSEUrlDeletedEvent, SSEMetadataUpdatedEvent } from '../types'

export interface DeletedNotification {
  id: string
  shortCode: string
  message: string
}

export const useUrlStore = defineStore('url', () => {
  const urls = ref<ShortURL[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref('')
  const deletedNotifications = ref<DeletedNotification[]>([])

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

  // Called by the SSE handler when the backend deletes a dead-link URL.
  function handleUrlDeleted(event: SSEUrlDeletedEvent) {
    urls.value = urls.value.filter((u) => u.id !== event.url_id)
    total.value = Math.max(0, total.value - 1)
    deletedNotifications.value.push({
      id: event.url_id,
      shortCode: event.short_code,
      message: `/${event.short_code} was removed — the original URL returned ${event.http_status || 'no response'}.`,
    })
  }

  function dismissNotification(id: string) {
    deletedNotifications.value = deletedNotifications.value.filter((n) => n.id !== id)
  }

  function handleMetadataUpdated(event: SSEMetadataUpdatedEvent) {
    const url = urls.value.find((u) => u.id === event.url_id)
    if (!url) return
    url.metadata = {
      title: event.title ?? null,
      description: event.description ?? null,
      og_image: event.og_image ?? null,
      favicon_url: event.favicon_url ?? null,
      fetch_status: event.fetch_status,
    }
  }

  function startSSE(token: string) {
    sseService.connect(token)
    sseService.onUrlDeleted(handleUrlDeleted)
    sseService.onMetadataUpdated(handleMetadataUpdated)
  }

  function stopSSE() {
    sseService.disconnect()
  }

  return {
    urls,
    total,
    loading,
    error,
    deletedNotifications,
    fetchAll,
    create,
    remove,
    handleUrlDeleted,
    handleMetadataUpdated,
    dismissNotification,
    startSSE,
    stopSSE,
  }
})
