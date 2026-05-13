<script setup lang="ts">
import { ref } from 'vue'
import { useUrlStore } from '../stores/urlStore'

const store = useUrlStore()
const copiedId = ref<string | null>(null)

async function copyLink(id: string, url: string) {
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = id
    setTimeout(() => {
      if (copiedId.value === id) copiedId.value = null
    }, 1500)
  } catch {
    // clipboard unavailable - silently skip
  }
}

async function handleDelete(id: string) {
  if (!confirm('Delete this shortened URL? This cannot be undone.')) return
  await store.remove(id)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
</script>

<template>
  <div class="url-list">
    <div v-if="store.loading" class="list-state">Loading...</div>

    <div v-else-if="store.urls.length === 0" class="list-state empty">
      <p>No shortened URLs yet.</p>
      <p class="hint">Paste a URL above to create your first short link.</p>
    </div>

    <div v-else class="url-table">
      <div class="url-row header">
        <div>Short URL</div>
        <div>Original</div>
        <div class="col-clicks">Clicks</div>
        <div class="col-date">Created</div>
        <div class="col-actions"></div>
      </div>

      <div v-for="u in store.urls" :key="u.id" class="url-row">
        <div class="col-short">
          <a :href="u.short_url" target="_blank" rel="noopener">
            {{ u.short_url }}
          </a>
        </div>
        <div class="col-original" :title="u.original_url">
          {{ u.original_url }}
        </div>
        <div class="col-clicks">{{ u.click_count }}</div>
        <div class="col-date">{{ formatDate(u.created_at) }}</div>
        <div class="col-actions">
          <button
            class="btn-icon"
            :class="{ copied: copiedId === u.id }"
            @click="copyLink(u.id, u.short_url)"
          >
            {{ copiedId === u.id ? 'Copied' : 'Copy' }}
          </button>
          <button class="btn-icon btn-danger" @click="handleDelete(u.id)">
            Delete
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.url-list {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #fff;
  overflow: hidden;
}

.list-state {
  padding: 2rem;
  text-align: center;
  color: #6b7280;
}

.list-state.empty p {
  margin: 0.25rem 0;
}

.list-state .hint {
  font-size: 0.9rem;
  color: #9ca3af;
}

.url-table {
  display: flex;
  flex-direction: column;
}

.url-row {
  display: grid;
  grid-template-columns: 1.4fr 2fr 80px 110px 160px;
  gap: 1rem;
  align-items: center;
  padding: 0.85rem 1.25rem;
  border-bottom: 1px solid #f3f4f6;
  font-size: 0.92rem;
}

.url-row:last-child {
  border-bottom: none;
}

.url-row.header {
  background: #f9fafb;
  font-weight: 600;
  color: #6b7280;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.col-short a {
  color: #4f46e5;
  text-decoration: none;
  font-weight: 500;
}

.col-short a:hover {
  text-decoration: underline;
}

.col-original {
  color: #374151;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-clicks {
  color: #374151;
  font-variant-numeric: tabular-nums;
}

.col-date {
  color: #6b7280;
}

.col-actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}

.btn-icon {
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  border: 1px solid #d1d5db;
  background: #fff;
  color: #374151;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon:hover {
  background: #f9fafb;
}

.btn-icon.copied {
  background: #d1fae5;
  border-color: #6ee7b7;
  color: #065f46;
}

.btn-icon.btn-danger {
  color: #b91c1c;
  border-color: #fecaca;
}

.btn-icon.btn-danger:hover {
  background: #fef2f2;
}

@media (max-width: 768px) {
  .url-row {
    grid-template-columns: 1fr;
    gap: 0.35rem;
  }
  .url-row.header {
    display: none;
  }
  .col-actions {
    justify-content: flex-start;
    margin-top: 0.25rem;
  }
}
</style>
