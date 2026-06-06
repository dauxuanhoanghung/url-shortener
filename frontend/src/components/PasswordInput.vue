<script setup lang="ts">
import { useVModel } from "@vueuse/core";
import { Eye, EyeOff } from "lucide-vue-next";
import { ref } from "vue";

import { Input } from "@/components/ui/input";

const props = defineProps<{
  modelValue?: string;
  id?: string;
  placeholder?: string;
  autocomplete?: string;
}>();

const emits = defineEmits<{
  (e: "update:modelValue", value: string): void;
}>();

const model = useVModel(props, "modelValue", emits, { passive: true, defaultValue: "" });
const visible = ref(false);
</script>

<template>
  <div class="relative">
    <Input
      :id="id"
      v-model="model"
      :type="visible ? 'text' : 'password'"
      :placeholder="placeholder"
      :autocomplete="autocomplete"
      class="pr-9"
    />
    <button
      type="button"
      tabindex="-1"
      :aria-label="visible ? 'Hide password' : 'Show password'"
      :aria-pressed="visible"
      class="text-muted-foreground hover:text-foreground absolute inset-y-0 right-0 flex items-center pr-2.5"
      @click="visible = !visible"
    >
      <EyeOff v-if="visible" class="h-4 w-4" />
      <Eye v-else class="h-4 w-4" />
    </button>
  </div>
</template>
