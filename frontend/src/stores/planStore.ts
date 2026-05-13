import { defineStore } from "pinia";
import { ref } from "vue";
import { planService } from "../services/planService";
import type { Plan } from "../types";

export const usePlanStore = defineStore("plans", () => {
  const plans = ref<Plan[]>([]);
  const loading = ref(false);
  const error = ref("");

  async function fetchAll() {
    if (plans.value.length > 0) return; // already loaded — avoid duplicate requests
    loading.value = true;
    error.value = "";
    try {
      const resp = await planService.list();
      if (resp.success && resp.data) {
        plans.value = resp.data;
      } else {
        error.value = resp.error?.message ?? "Failed to load plans";
      }
    } catch (err: any) {
      error.value =
        err.response?.data?.error?.message ?? "Failed to load plans";
    } finally {
      loading.value = false;
    }
  }

  return { plans, loading, error, fetchAll };
});
