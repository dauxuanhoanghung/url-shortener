<script setup lang="ts">
import { ref } from 'vue'
import { useUrlStore } from '@/stores/urlStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const store = useUrlStore()
const originalUrl = ref('')
const submitting = ref(false)
const localError = ref('')

async function handleSubmit() {
  localError.value = ''
  if (!originalUrl.value.trim()) return

  submitting.value = true
  try {
    await store.create(originalUrl.value.trim())
    originalUrl.value = ''
  } catch {
    localError.value = store.error
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="mb-6" @submit.prevent="handleSubmit">
    <div class="flex gap-2">
      <Input
        v-model="originalUrl"
        type="url"
        placeholder="Paste a long URL, e.g. https://example.com/some/long/path"
        required
        :disabled="submitting"
        class="flex-1"
      />
      <Button type="submit" :disabled="submitting" class="shrink-0">
        {{ submitting ? 'Shortening…' : 'Shorten' }}
      </Button>
    </div>
    <p v-if="localError" class="mt-2 text-xs text-destructive">{{ localError }}</p>
  </form>
</template>
