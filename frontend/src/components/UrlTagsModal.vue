<script setup lang="ts">
import { ref, watch } from "vue";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText,
} from "@/components/ui/tags-input";
import { useUrlStore } from "@/stores/urlStore";
import type { ShortURL } from "@/types";

const props = defineProps<{ open: boolean; url: ShortURL }>();
const emit = defineEmits<{ (e: "update:open", value: boolean): void }>();

const store = useUrlStore();
const tags = ref<string[]>([]);
const saving = ref(false);
const serverError = ref("");

watch(
  () => props.url,
  (url) => {
    tags.value = [...(url?.tags ?? [])];
  },
  { immediate: true },
);

async function save() {
  saving.value = true;
  serverError.value = "";
  try {
    await store.updateTags(props.url.id, tags.value);
    emit("update:open", false);
  } catch (err: any) {
    serverError.value = err.response?.data?.error?.message || "Failed to save tags";
  } finally {
    saving.value = false;
  }
}

function onOpenChange(val: boolean) {
  if (!val) serverError.value = "";
  emit("update:open", val);
}
</script>

<template>
  <Dialog :open="props.open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Edit tags</DialogTitle>
      </DialogHeader>

      <div class="space-y-4">
        <Alert v-if="serverError" variant="destructive">
          <AlertDescription>{{ serverError }}</AlertDescription>
        </Alert>

        <div class="space-y-1.5">
          <Label>Tags</Label>
          <TagsInput v-model="tags">
            <TagsInputItem v-for="tag in tags" :key="tag" :value="tag">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="Add tag, press Enter" />
          </TagsInput>
          <p class="text-muted-foreground text-xs">Press Enter or comma to add · max 20 tags</p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="onOpenChange(false)">Cancel</Button>
        <Button :disabled="saving" @click="save">
          {{ saving ? "Saving…" : "Save tags" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
