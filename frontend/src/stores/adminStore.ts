import { defineStore } from "pinia";
import { ref } from "vue";
import { adminService } from "../services/adminService";
import type { AdminAuditEntry, AdminUser } from "../types";

export const useAdminStore = defineStore("admin", () => {
  const users = ref<AdminUser[]>([]);
  const userTotal = ref(0);
  const auditEntries = ref<AdminAuditEntry[]>([]);
  const auditTotal = ref(0);
  const loading = ref(false);
  const error = ref("");

  async function fetchUsers(limit = 50, offset = 0) {
    loading.value = true;
    error.value = "";
    try {
      const resp = await adminService.listUsers(limit, offset);
      if (resp.success && resp.data) {
        users.value = resp.data.users;
        userTotal.value = resp.data.total;
      } else {
        error.value = resp.error?.message ?? "Failed to load users";
      }
    } catch (err: any) {
      error.value =
        err.response?.data?.error?.message ?? "Failed to load users";
    } finally {
      loading.value = false;
    }
  }

  async function setUserDisabled(id: string, disabled: boolean) {
    error.value = "";
    try {
      const resp = disabled
        ? await adminService.disableUser(id)
        : await adminService.enableUser(id);
      if (resp.success && resp.data) {
        const idx = users.value.findIndex((u) => u.id === id);
        if (idx >= 0) users.value[idx] = resp.data;
      } else {
        error.value = resp.error?.message ?? "Failed to update user";
      }
    } catch (err: any) {
      error.value =
        err.response?.data?.error?.message ?? "Failed to update user";
    }
  }

  async function fetchAudit(limit = 50, offset = 0) {
    loading.value = true;
    error.value = "";
    try {
      const resp = await adminService.listAudit(limit, offset);
      if (resp.success && resp.data) {
        auditEntries.value = resp.data.entries;
        auditTotal.value = resp.data.total;
      } else {
        error.value = resp.error?.message ?? "Failed to load audit log";
      }
    } catch (err: any) {
      error.value =
        err.response?.data?.error?.message ?? "Failed to load audit log";
    } finally {
      loading.value = false;
    }
  }

  return {
    users,
    userTotal,
    auditEntries,
    auditTotal,
    loading,
    error,
    fetchUsers,
    setUserDisabled,
    fetchAudit,
  };
});
