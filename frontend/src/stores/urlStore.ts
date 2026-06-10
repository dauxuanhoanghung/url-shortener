import { defineStore } from "pinia";
import { ref } from "vue";

import { urlService, type URLListParams } from "../services/urlService";
import type { ShortURL, SSEMetadataUpdatedEvent, SSEUrlDeletedEvent } from "../types";

export interface DeletedNotification {
  id: string;
  shortCode: string;
  message: string;
}

export const useUrlStore = defineStore("url", () => {
  const urls = ref<ShortURL[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const error = ref("");
  const deletedNotifications = ref<DeletedNotification[]>([]);
  const filters = ref<URLListParams>({});

  async function fetchAll() {
    loading.value = true;
    error.value = "";
    try {
      const resp = await urlService.list(filters.value);
      if (resp.success && resp.data) {
        urls.value = resp.data.urls;
        total.value = resp.data.total;
      }
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || "Failed to load URLs";
    } finally {
      loading.value = false;
    }
  }

  function setFilter(patch: Partial<URLListParams>) {
    filters.value = { ...filters.value, ...patch };
  }

  function resetFilters() {
    filters.value = {};
  }

  async function create(originalUrl: string, tags: string[] = []): Promise<ShortURL | null> {
    error.value = "";
    try {
      const resp = await urlService.create(originalUrl, tags);
      if (resp.success && resp.data) {
        urls.value = [resp.data, ...urls.value];
        total.value += 1;
        return resp.data;
      }
      return null;
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || "Failed to create URL";
      throw err;
    }
  }

  async function updateTags(id: string, tags: string[]): Promise<void> {
    error.value = "";
    try {
      const resp = await urlService.updateTags(id, tags);
      if (resp.success && resp.data) {
        const idx = urls.value.findIndex((u) => u.id === id);
        if (idx !== -1) {
          urls.value[idx] = { ...urls.value[idx], tags: resp.data.tags };
        }
      }
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || "Failed to update tags";
      throw err;
    }
  }

  async function remove(id: string) {
    error.value = "";
    try {
      const resp = await urlService.remove(id);
      if (resp.success) {
        urls.value = urls.value.filter((u) => u.id !== id);
        total.value = Math.max(0, total.value - 1);
      }
    } catch (err: any) {
      error.value = err.response?.data?.error?.message || "Failed to delete URL";
      throw err;
    }
  }

  function handleUrlDeleted(event: SSEUrlDeletedEvent) {
    urls.value = urls.value.filter((u) => u.id !== event.url_id);
    total.value = Math.max(0, total.value - 1);
    deletedNotifications.value.push({
      id: event.url_id,
      shortCode: event.short_code,
      message: `/${event.short_code} was removed — the original URL returned ${event.http_status || "no response"}.`,
    });
  }

  function handleMetadataUpdated(event: SSEMetadataUpdatedEvent) {
    const url = urls.value.find((u) => u.id === event.url_id);
    if (!url) return;
    url.metadata = {
      title: event.title ?? null,
      description: event.description ?? null,
      og_image: event.og_image ?? null,
      favicon_url: event.favicon_url ?? null,
      fetch_status: event.fetch_status,
    };
  }

  function dismissNotification(id: string) {
    deletedNotifications.value = deletedNotifications.value.filter((n) => n.id !== id);
  }

  return {
    urls,
    total,
    loading,
    error,
    filters,
    deletedNotifications,
    fetchAll,
    setFilter,
    resetFilters,
    create,
    updateTags,
    remove,
    handleUrlDeleted,
    handleMetadataUpdated,
    dismissNotification,
  };
});
