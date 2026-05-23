<script setup lang="ts">
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAdminStore } from "@/stores/adminStore";
import { useAuthStore } from "@/stores/authStore";
import { onMounted } from "vue";

const store = useAdminStore();
const auth = useAuthStore();

onMounted(() => store.fetchUsers());

async function toggleDisabled(id: string, currentlyDisabled: boolean) {
  await store.setUserDisabled(id, !currentlyDisabled);
}

function formatDate(s: string) {
  return new Date(s).toLocaleString();
}
</script>

<template>
  <section>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold">Users ({{ store.userTotal }})</h2>
      <Button variant="outline" size="sm" @click="store.fetchUsers()" :disabled="store.loading">
        Refresh
      </Button>
    </div>

    <Alert v-if="store.error" variant="destructive" class="mb-4">
      <AlertDescription>{{ store.error }}</AlertDescription>
    </Alert>

    <p v-if="store.loading" class="text-sm text-muted-foreground">Loading…</p>

    <div v-else class="overflow-x-auto rounded-md border">
      <table class="w-full text-sm">
        <thead class="border-b bg-muted/40 text-left">
          <tr>
            <th class="px-3 py-2 font-medium">Email</th>
            <th class="px-3 py-2 font-medium">Role</th>
            <th class="px-3 py-2 font-medium">Plan</th>
            <th class="px-3 py-2 font-medium">Verified</th>
            <th class="px-3 py-2 font-medium">Status</th>
            <th class="px-3 py-2 font-medium">Created</th>
            <th class="px-3 py-2 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in store.users" :key="u.id" class="border-b last:border-0 hover:bg-muted/20">
            <td class="px-3 py-2">{{ u.email }}</td>
            <td class="px-3 py-2">
              <Badge :variant="u.role === 'admin' ? 'default' : 'outline'">{{ u.role }}</Badge>
            </td>
            <td class="px-3 py-2">{{ u.plan_code || "—" }}</td>
            <td class="px-3 py-2">
              <span v-if="u.email_verified" class="text-emerald-600">✓</span>
              <span v-else class="text-muted-foreground">pending</span>
            </td>
            <td class="px-3 py-2">
              <Badge v-if="u.disabled" variant="destructive">disabled</Badge>
              <Badge v-else variant="outline" class="border-emerald-500 text-emerald-700">active</Badge>
            </td>
            <td class="px-3 py-2 text-muted-foreground">{{ formatDate(u.created_at) }}</td>
            <td class="px-3 py-2 text-right">
              <Button
                size="sm"
                :variant="u.disabled ? 'outline' : 'destructive'"
                :disabled="u.id === auth.user?.id"
                @click="toggleDisabled(u.id, u.disabled)"
              >
                {{ u.disabled ? "Enable" : "Disable" }}
              </Button>
            </td>
          </tr>
          <tr v-if="store.users.length === 0">
            <td colspan="7" class="px-3 py-6 text-center text-muted-foreground">No users.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
