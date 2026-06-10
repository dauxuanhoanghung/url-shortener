<script setup lang="ts">
import { PencilIcon } from "lucide-vue-next";
import { ref } from "vue";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import UrlTagsModal from "@/components/UrlTagsModal.vue";
import { useUrlStore } from "@/stores/urlStore";
import type { ShortURL } from "@/types";

const store = useUrlStore();
const copiedId = ref<string | null>(null);
// Track URLs whose og_image failed to load so we don't show the strip.
// Keyed by url.id + og_image to reset when the image src changes via SSE.
const brokenOgImages = ref(new Set<string>());
const editingUrl = ref<ShortURL | null>(null);
const isTagsModalOpen = ref(false);

function ogImageKey(u: ShortURL) {
  return `${u.id}::${u.metadata?.og_image ?? ""}`;
}

function onOgError(u: ShortURL) {
  brokenOgImages.value.add(ogImageKey(u));
}

function showOgImage(u: ShortURL) {
  return !!u.metadata?.og_image && !brokenOgImages.value.has(ogImageKey(u));
}

async function copyLink(id: string, url: string) {
  try {
    await navigator.clipboard.writeText(url);
    copiedId.value = id;
    setTimeout(() => {
      if (copiedId.value === id) copiedId.value = null;
    }, 1500);
  } catch {
    // clipboard unavailable — silently skip
  }
}

async function handleDelete(id: string) {
  if (!confirm("Delete this shortened URL? This cannot be undone.")) return;
  await store.remove(id);
}

function openTagsModal(u: ShortURL) {
  editingUrl.value = u;
  isTagsModalOpen.value = true;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function shortHost(url: string) {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

function faviconSrc(u: ShortURL): string | null {
  if (u.metadata?.favicon_url) return u.metadata.favicon_url;
  try {
    const { origin } = new URL(u.original_url);
    return `${origin}/favicon.ico`;
  } catch {
    return null;
  }
}
</script>

<template>
  <div v-if="store.loading" class="text-muted-foreground py-8 text-center text-sm">Loading…</div>

  <div
    v-else-if="store.urls.length === 0"
    class="rounded-lg border border-dashed py-12 text-center"
  >
    <p class="text-foreground text-sm font-medium">No shortened URLs yet.</p>
    <p class="text-muted-foreground mt-1 text-xs">
      Click "Shorten URL" above to create your first short link.
    </p>
  </div>

  <div v-else class="flex flex-col gap-3">
    <div
      v-for="u in store.urls"
      :key="u.id"
      class="border-border bg-card rounded-lg border shadow-sm"
    >
      <!-- Main row -->
      <div class="flex items-start gap-3 p-4">
        <!-- Favicon -->
        <div
          class="bg-muted mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md border"
        >
          <img
            v-if="faviconSrc(u)"
            :src="faviconSrc(u)!"
            :alt="shortHost(u.original_url)"
            class="h-5 w-5 object-contain"
            @error="($event.target as HTMLImageElement).style.display = 'none'"
          />
          <span v-else class="text-muted-foreground text-xs">🔗</span>
        </div>

        <!-- URL info -->
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <a
              :href="u.short_url"
              target="_blank"
              rel="noopener"
              class="text-primary text-sm font-semibold hover:underline"
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
              class="border-red-200 bg-red-50 text-xs text-red-600"
            >
              Unreachable
            </Badge>
          </div>

          <!-- Page title from metadata, or fallback to original URL -->
          <p
            v-if="u.metadata?.title"
            class="text-foreground mt-0.5 truncate text-sm font-medium"
            :title="u.metadata.title"
          >
            {{ u.metadata.title }}
          </p>
          <p class="text-muted-foreground mt-0.5 truncate text-xs" :title="u.original_url">
            {{ u.original_url }}
          </p>

          <!-- Description -->
          <p v-if="u.metadata?.description" class="text-muted-foreground mt-1 line-clamp-2 text-xs">
            {{ u.metadata.description }}
          </p>

          <!-- Tags -->
          <div v-if="u.tags && u.tags.length > 0" class="mt-2 flex flex-wrap gap-1">
            <Badge v-for="tag in u.tags" :key="tag" variant="secondary" class="text-xs font-normal">
              {{ tag }}
            </Badge>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex shrink-0 flex-col items-end gap-1.5">
          <div class="flex gap-1.5">
            <Button
              size="sm"
              :variant="copiedId === u.id ? 'secondary' : 'outline'"
              class="h-7 px-2.5 text-xs"
              :class="copiedId === u.id ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : ''"
              @click="copyLink(u.id, u.short_url)"
            >
              {{ copiedId === u.id ? "Copied!" : "Copy" }}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              class="h-7 px-2 text-xs"
              title="Edit tags"
              @click="openTagsModal(u)"
            >
              <PencilIcon class="h-3.5 w-3.5" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              class="text-destructive hover:bg-destructive/10 hover:text-destructive h-7 px-2.5 text-xs"
              @click="handleDelete(u.id)"
            >
              Delete
            </Button>
          </div>
          <div class="text-muted-foreground flex items-center gap-2 text-xs">
            <span>{{ u.click_count }} click{{ u.click_count !== 1 ? "s" : "" }}</span>
            <span>·</span>
            <span>{{ formatDate(u.created_at) }}</span>
          </div>
        </div>
      </div>

      <!-- OG image strip — only shown when available and not broken -->
      <div v-if="showOgImage(u)" class="border-t px-4 pb-3 pt-2">
        <img
          :src="u.metadata!.og_image!"
          :alt="u.metadata!.title || shortHost(u.original_url)"
          class="h-32 w-full rounded object-cover"
          @error="onOgError(u)"
        />
      </div>
    </div>
  </div>

  <UrlTagsModal
    v-if="editingUrl"
    :open="isTagsModalOpen"
    :url="editingUrl"
    @update:open="isTagsModalOpen = $event"
  />
</template>
