<script setup lang="ts">
import { toTypedSchema } from "@vee-validate/zod";
import { useForm } from "vee-validate";
import { ref } from "vue";
import * as z from "zod";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText,
} from "@/components/ui/tags-input";
import { useUrlStore } from "@/stores/urlStore";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ (e: "update:open", value: boolean): void }>();

const store = useUrlStore();
const serverError = ref("");
const tags = ref<string[]>([]);

const schema = toTypedSchema(
  z.object({
    originalUrl: z.string().min(1, "URL is required").url("Must be a valid URL"),
  }),
);

const { defineField, handleSubmit, errors, isSubmitting, resetForm } = useForm({
  validationSchema: schema,
  initialValues: { originalUrl: "" },
});

const [originalUrl, originalUrlAttrs] = defineField("originalUrl");

const onSubmit = handleSubmit(async (values) => {
  serverError.value = "";
  try {
    await store.create(values.originalUrl, tags.value);
    resetForm();
    tags.value = [];
    emit("update:open", false);
  } catch (err: any) {
    serverError.value = err.response?.data?.error?.message || "Failed to create URL";
  }
});

function onOpenChange(val: boolean) {
  if (!val) {
    resetForm();
    tags.value = [];
    serverError.value = "";
  }
  emit("update:open", val);
}
</script>

<template>
  <Dialog :open="props.open" @update:open="onOpenChange">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Shorten a URL</DialogTitle>
      </DialogHeader>

      <form novalidate class="space-y-4" @submit="onSubmit">
        <Alert v-if="serverError" variant="destructive">
          <AlertDescription>{{ serverError }}</AlertDescription>
        </Alert>

        <div class="space-y-1.5">
          <Label for="originalUrl">Destination URL</Label>
          <Input
            id="originalUrl"
            v-model="originalUrl"
            v-bind="originalUrlAttrs"
            placeholder="https://example.com/long-url"
            :aria-invalid="!!errors.originalUrl"
          />
          <p v-if="errors.originalUrl" class="text-destructive text-sm">
            {{ errors.originalUrl }}
          </p>
        </div>

        <div class="space-y-1.5">
          <Label>Tags <span class="text-muted-foreground text-xs">(optional)</span></Label>
          <TagsInput v-model="tags">
            <TagsInputItem v-for="tag in tags" :key="tag" :value="tag">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="Add tag, press Enter" />
          </TagsInput>
          <p class="text-muted-foreground text-xs">Press Enter or comma to add a tag</p>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" @click="onOpenChange(false)">Cancel</Button>
          <Button type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? "Shortening…" : "Shorten URL" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
