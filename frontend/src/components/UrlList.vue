<script setup lang="ts">
import { ref } from 'vue'
import { useUrlStore } from '@/stores/urlStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

const store = useUrlStore()
const copiedId = ref<string | null>(null)

async function copyLink(id: string, url: string) {
  try {
    await navigator.clipboard.writeText(url)
    copiedId.value = id
    setTimeout(() => { if (copiedId.value === id) copiedId.value = null }, 1500)
  } catch {
    // clipboard unavailable — silently skip
  }
}

async function handleDelete(id: string) {
  if (!confirm('Delete this shortened URL? This cannot be undone.')) return
  await store.remove(id)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <div v-if="store.loading" class="py-8 text-center text-sm text-muted-foreground">Loading…</div>

  <div v-else-if="store.urls.length === 0" class="rounded-lg border border-dashed py-12 text-center">
    <p class="text-sm font-medium text-foreground">No shortened URLs yet.</p>
    <p class="mt-1 text-xs text-muted-foreground">Paste a URL above to create your first short link.</p>
  </div>

  <div v-else class="overflow-hidden rounded-lg border border-border">
    <!-- Table header — hidden on mobile -->
    <div class="hidden grid-cols-[1.4fr_2fr_80px_110px_150px] gap-4 border-b bg-muted/50 px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground md:grid">
      <div>Short URL</div>
      <div>Original</div>
      <div>Clicks</div>
      <div>Created</div>
      <div></div>
    </div>

    <div
      v-for="u in store.urls"
      :key="u.id"
      class="flex flex-col gap-1.5 border-b px-4 py-3 last:border-0 md:grid md:grid-cols-[1.4fr_2fr_80px_110px_150px] md:items-center md:gap-4"
    >
      <div>
        <a :href="u.short_url" target="_blank" rel="noopener" class="text-sm font-medium text-primary hover:underline">
          {{ u.short_url }}
        </a>
      </div>

      <div class="max-w-full truncate text-sm text-muted-foreground" :title="u.original_url">
        {{ u.original_url }}
      </div>

      <div>
        <Badge variant="secondary" class="font-mono text-xs">{{ u.click_count }}</Badge>
      </div>

      <div class="text-xs text-muted-foreground">{{ formatDate(u.created_at) }}</div>

      <div class="flex gap-2 md:justify-end">
        <Button
          size="sm"
          :variant="copiedId === u.id ? 'secondary' : 'outline'"
          class="h-7 px-2.5 text-xs"
          :class="copiedId === u.id ? 'bg-emerald-50 text-emerald-700 border-emerald-200' : ''"
          @click="copyLink(u.id, u.short_url)"
        >
          {{ copiedId === u.id ? 'Copied!' : 'Copy' }}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          class="h-7 px-2.5 text-xs text-destructive hover:bg-destructive/10 hover:text-destructive"
          @click="handleDelete(u.id)"
        >
          Delete
        </Button>
      </div>
    </div>
  </div>
</template>
