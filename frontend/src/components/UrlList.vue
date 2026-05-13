<script setup lang="ts">
import { ref } from 'vue'
import { useUrlStore } from '@/stores/urlStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import type { ShortURL } from '@/types'

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

function shortHost(url: string) {
  try { return new URL(url).hostname } catch { return url }
}

function faviconSrc(u: ShortURL): string | null {
  if (u.metadata?.favicon_url) return u.metadata.favicon_url
  try {
    const { origin } = new URL(u.original_url)
    return `${origin}/favicon.ico`
  } catch {
    return null
  }
}
</script>

<template>
  <div v-if="store.loading" class="py-8 text-center text-sm text-muted-foreground">Loading…</div>

  <div v-else-if="store.urls.length === 0" class="rounded-lg border border-dashed py-12 text-center">
    <p class="text-sm font-medium text-foreground">No shortened URLs yet.</p>
    <p class="mt-1 text-xs text-muted-foreground">Paste a URL above to create your first short link.</p>
  </div>

  <div v-else class="flex flex-col gap-3">
    <div
      v-for="u in store.urls"
      :key="u.id"
      class="rounded-lg border border-border bg-card shadow-sm"
    >
      <!-- Main row -->
      <div class="flex items-start gap-3 p-4">

        <!-- Favicon -->
        <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md border bg-muted">
          <img
            v-if="faviconSrc(u)"
            :src="faviconSrc(u)!"
            :alt="shortHost(u.original_url)"
            class="h-5 w-5 object-contain"
            @error="($event.target as HTMLImageElement).style.display='none'"
          />
          <span v-else class="text-xs text-muted-foreground">🔗</span>
        </div>

        <!-- URL info -->
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <a
              :href="u.short_url"
              target="_blank"
              rel="noopener"
              class="text-sm font-semibold text-primary hover:underline"
            >
              {{ u.short_url }}
            </a>

            <!-- Fetch status badge -->
            <Badge
              v-if="u.metadata?.fetch_status === 'pending'"
              variant="secondary"
              class="text-xs"
            >
              Fetching…
            </Badge>
            <Badge
              v-else-if="u.metadata?.fetch_status === 'failed'"
              class="bg-red-50 text-red-600 border-red-200 text-xs"
            >
              Unreachable
            </Badge>
          </div>

          <!-- Page title from metadata, or fallback to original URL -->
          <p
            v-if="u.metadata?.title"
            class="mt-0.5 truncate text-sm font-medium text-foreground"
            :title="u.metadata.title"
          >
            {{ u.metadata.title }}
          </p>
          <p
            class="mt-0.5 truncate text-xs text-muted-foreground"
            :title="u.original_url"
          >
            {{ u.original_url }}
          </p>

          <!-- Description -->
          <p
            v-if="u.metadata?.description"
            class="mt-1 line-clamp-2 text-xs text-muted-foreground"
          >
            {{ u.metadata.description }}
          </p>
        </div>

        <!-- Actions -->
        <div class="flex shrink-0 flex-col items-end gap-1.5">
          <div class="flex gap-1.5">
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
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{{ u.click_count }} click{{ u.click_count !== 1 ? 's' : '' }}</span>
            <span>·</span>
            <span>{{ formatDate(u.created_at) }}</span>
          </div>
        </div>
      </div>

      <!-- OG image strip — only shown when available -->
      <div
        v-if="u.metadata?.og_image"
        class="border-t px-4 pb-3 pt-2"
      >
        <img
          :src="u.metadata.og_image"
          :alt="u.metadata.title || shortHost(u.original_url)"
          class="h-32 w-full rounded object-cover"
          @error="($event.target as HTMLImageElement).parentElement!.style.display='none'"
        />
      </div>
    </div>
  </div>
</template>
