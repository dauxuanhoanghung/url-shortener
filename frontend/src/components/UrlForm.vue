<script setup lang="ts">
import { ref } from 'vue'
import { useUrlStore } from '../stores/urlStore'

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
  <form class="url-form" @submit.prevent="handleSubmit">
    <div class="url-form-row">
      <input
        v-model="originalUrl"
        type="url"
        placeholder="Paste a long URL, e.g. https://example.com/some/long/path"
        required
        :disabled="submitting"
      />
      <button type="submit" class="btn btn-primary" :disabled="submitting">
        {{ submitting ? 'Shortening...' : 'Shorten' }}
      </button>
    </div>
    <p v-if="localError" class="form-error">{{ localError }}</p>
  </form>
</template>

<style scoped>
.url-form {
  margin-bottom: 2rem;
}

.url-form-row {
  display: flex;
  gap: 0.75rem;
}

.url-form-row input {
  flex: 1;
  padding: 0.7rem 0.9rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 1rem;
  outline: none;
  transition: border-color 0.15s;
}

.url-form-row input:focus {
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px #4f46e520;
}

.btn {
  padding: 0.7rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  border: none;
  white-space: nowrap;
}

.btn-primary {
  background: #4f46e5;
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  background: #4338ca;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-error {
  margin-top: 0.5rem;
  color: #991b1b;
  font-size: 0.9rem;
}
</style>
