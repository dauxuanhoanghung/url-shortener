<script setup lang="ts">
import { Card } from "@/components/ui/card";
</script>

<template>
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <router-view />
    </Card>
  </div>
</template>
