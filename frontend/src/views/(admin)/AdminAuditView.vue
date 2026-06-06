<script setup lang="ts">
import { onMounted } from "vue";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAdminStore } from "@/stores/adminStore";

const store = useAdminStore();

onMounted(() => store.fetchAudit());

function formatDate(s: string) {
  return new Date(s).toLocaleString();
}

function formatJson(obj: Record<string, any> | undefined) {
  if (!obj) return "—";
  return JSON.stringify(obj);
}
</script>

<template>
  <section>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold">Audit log ({{ store.auditTotal }})</h2>
      <Button variant="outline" size="sm" @click="store.fetchAudit()" :disabled="store.loading">
        Refresh
      </Button>
    </div>

    <Alert v-if="store.error" variant="destructive" class="mb-4">
      <AlertDescription>{{ store.error }}</AlertDescription>
    </Alert>

    <p v-if="store.loading" class="text-muted-foreground text-sm">Loading…</p>

    <div v-else class="overflow-x-auto rounded-md border">
      <table class="w-full text-sm">
        <thead class="bg-muted/40 border-b text-left">
          <tr>
            <th class="px-3 py-2 font-medium">When</th>
            <th class="px-3 py-2 font-medium">Action</th>
            <th class="px-3 py-2 font-medium">Target</th>
            <th class="px-3 py-2 font-medium">Actor</th>
            <th class="px-3 py-2 font-medium">Before</th>
            <th class="px-3 py-2 font-medium">After</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in store.auditEntries" :key="e.id" class="border-b align-top last:border-0">
            <td class="text-muted-foreground whitespace-nowrap px-3 py-2">
              {{ formatDate(e.created_at) }}
            </td>
            <td class="px-3 py-2">
              <Badge variant="outline">{{ e.action }}</Badge>
            </td>
            <td class="px-3 py-2">
              <span class="text-muted-foreground">{{ e.target_type }}</span>
              <div class="font-mono text-xs">{{ e.target_id }}</div>
            </td>
            <td class="px-3 py-2 font-mono text-xs">
              {{ e.actor_id ?? "(cli)" }}
            </td>
            <td class="text-muted-foreground px-3 py-2 text-xs">{{ formatJson(e.before) }}</td>
            <td class="px-3 py-2 text-xs">{{ formatJson(e.after) }}</td>
          </tr>
          <tr v-if="store.auditEntries.length === 0">
            <td colspan="6" class="text-muted-foreground px-3 py-6 text-center">
              No audit entries yet.
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
